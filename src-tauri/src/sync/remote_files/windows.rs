//! Windows 虚拟文件剪贴板。
//!
//! 复制端文件不预传：这里在 OLE 剪贴板上放一个自实现的 `IDataObject`，只提供
//! `FileGroupDescriptorW`（清单）与按需的 `FileContents`（每个文件一个 `IStream`）。
//! 目标应用（资源管理器、聊天软件等）粘贴时才会读取流，流的 `Read` 再经同步会话向对端拉取数据。
//!
//! 线程模型：每次 [`present`] 起一个 STA 线程，为该剪贴板对象及其派生的文件流泵消息。
//! 新复制可以替换系统剪贴板；已启动的粘贴仍会持有旧对象或旧文件流。

use std::collections::VecDeque;
use std::ffi::c_void;
use std::mem::ManuallyDrop;
use std::sync::{Arc, Mutex};

use tauri::AppHandle;
use windows::core::{implement, Error, Result as WinResult, HRESULT, PCWSTR};
use windows::Win32::Foundation::{
    BOOL, DATA_S_SAMEFORMATETC, DV_E_FORMATETC, DV_E_LINDEX, DV_E_TYMED, E_NOTIMPL, FALSE,
    FILETIME, OLE_E_ADVISENOTSUPPORTED, STG_E_INVALIDFUNCTION, STG_E_READFAULT, S_OK,
};
use windows::Win32::System::Com::{
    IAdviseSink, IDataObject, IDataObject_Impl, IEnumFORMATETC, IEnumSTATDATA,
    ISequentialStream_Impl, IStream, IStream_Impl, DATADIR_GET, DVASPECT_CONTENT, FORMATETC,
    STATFLAG, STATSTG, STGC, STGMEDIUM, STGMEDIUM_0, STGTY_STREAM, STREAM_SEEK, STREAM_SEEK_CUR,
    STREAM_SEEK_END, STREAM_SEEK_SET, TYMED_HGLOBAL, TYMED_ISTREAM,
};
use windows::Win32::System::Com::{LOCKTYPE, STGM};
use windows::Win32::System::DataExchange::RegisterClipboardFormatW;
use windows::Win32::System::Memory::{GlobalAlloc, GlobalLock, GlobalUnlock, GMEM_MOVEABLE};
use windows::Win32::System::Ole::{OleInitialize, OleSetClipboard, OleUninitialize};
use windows::Win32::UI::Shell::{
    SHCreateStdEnumFmtEtc, FD_ATTRIBUTES, FD_FILESIZE, FD_PROGRESSUI, FILEDESCRIPTORW,
};
use windows::Win32::UI::WindowsAndMessaging::{
    DispatchMessageW, MsgWaitForMultipleObjects, PeekMessageW, TranslateMessage, MSG, PM_REMOVE,
    QS_ALLINPUT,
};

use crate::sync::manager::{fetch_file_range, SyncState};
use crate::sync::protocol::{FileEntry, CHUNK_SIZE};

const FILE_ATTRIBUTE_DIRECTORY: u32 = 0x10;
const FILE_ATTRIBUTE_NORMAL: u32 = 0x80;
const MAX_NAME_UNITS: usize = 259;
const CLIPBOARD_POLL: u32 = 300;

fn wide(text: &str) -> Vec<u16> {
    text.encode_utf16().chain(std::iter::once(0)).collect()
}

fn register_format(name: &str) -> u16 {
    let name = wide(name);
    unsafe { RegisterClipboardFormatW(PCWSTR(name.as_ptr())) as u16 }
}

/// 对端文件清单在本机剪贴板上的虚拟呈现。
pub fn present(
    _app: &AppHandle,
    state: SyncState,
    peer_id: &str,
    offer: &str,
    entries: Vec<FileEntry>,
) {
    let peer_id = peer_id.to_owned();
    let offer = offer.to_owned();

    let spawned = std::thread::Builder::new()
        .name("sync-virtual-files".to_owned())
        .spawn(move || {
            if let Err(err) = run_clipboard_thread(state, peer_id, offer, entries) {
                log::warn!("sync: virtual file clipboard failed: {err}");
            }
        });
    if let Err(err) = spawned {
        log::warn!("sync: cannot start virtual file thread: {err}");
    }
}

/// 构造虚拟文件数据对象和 STA 的存活条件。派生的文件流与数据对象共同持有文件来源。
pub fn create_data_object(
    state: SyncState,
    peer_id: String,
    offer: String,
    entries: Vec<FileEntry>,
) -> (IDataObject, impl Fn() -> bool) {
    let offer = Arc::new(RemoteOffer {
        state,
        peer_id,
        offer,
        entries,
    });
    let lifetime = Arc::downgrade(&offer);
    (offer.data_object(), move || lifetime.strong_count() != 0)
}

fn run_clipboard_thread(
    state: SyncState,
    peer_id: String,
    offer: String,
    entries: Vec<FileEntry>,
) -> WinResult<()> {
    unsafe {
        OleInitialize(None)?;
    }

    let result = (|| -> WinResult<()> {
        let (object, is_in_use) = create_data_object(state, peer_id, offer, entries);

        set_clipboard_with_retry(&object)?;
        log::info!("sync: virtual files placed on the clipboard");

        // OleSetClipboard 持有自己的 COM 引用。释放本地引用，使对象和派生流的实际持有者决定 STA 的寿命。
        drop(object);
        pump_while(is_in_use);
        Ok(())
    })();

    unsafe {
        OleUninitialize();
    }
    result
}

/// 远程桌面等程序会长时间占用剪贴板，`OleSetClipboard` 因此返回 `CLIPBRD_E_CANT_OPEN`；
/// 退避重试约 15 秒，期间泵消息保持 STA 响应。
fn set_clipboard_with_retry(object: &IDataObject) -> WinResult<()> {
    const MAX_ATTEMPTS: u32 = 40;
    let mut delay = std::time::Duration::from_millis(30);

    for attempt in 1..=MAX_ATTEMPTS {
        match unsafe { OleSetClipboard(object) } {
            Ok(()) => return Ok(()),
            Err(err) if attempt == MAX_ATTEMPTS => return Err(err),
            Err(err) => {
                log::debug!("sync: clipboard busy for virtual files (attempt {attempt}): {err}");
                std::thread::sleep(delay);
                delay = (delay * 2).min(std::time::Duration::from_millis(500));
            }
        }
    }
    Ok(())
}

/// 持有剪贴板对象或派生流的应用仍可能发出 COM 调用，即使剪贴板已被替换。
fn pump_while(is_in_use: impl Fn() -> bool) {
    while is_in_use() {
        unsafe {
            let mut msg = MSG::default();
            while PeekMessageW(&mut msg, None, 0, 0, PM_REMOVE).as_bool() {
                let _ = TranslateMessage(&msg);
                DispatchMessageW(&msg);
            }

            MsgWaitForMultipleObjects(None, FALSE, CLIPBOARD_POLL, QS_ALLINPUT);
        }
    }
}

struct RemoteOffer {
    state: SyncState,
    peer_id: String,
    offer: String,
    entries: Vec<FileEntry>,
}

impl RemoteOffer {
    fn data_object(self: &Arc<Self>) -> IDataObject {
        VirtualFiles {
            offer: self.clone(),
            fmt_descriptor: register_format("FileGroupDescriptorW"),
            fmt_contents: register_format("FileContents"),
        }
        .into()
    }
}

#[implement(IDataObject)]
struct VirtualFiles {
    offer: Arc<RemoteOffer>,
    fmt_descriptor: u16,
    fmt_contents: u16,
}

impl VirtualFiles {
    /// 构造 `FILEGROUPDESCRIPTORW`：`u32` 个数 + 逐项 `FILEDESCRIPTORW`。
    fn build_descriptor(&self) -> WinResult<STGMEDIUM> {
        let item = std::mem::size_of::<FILEDESCRIPTORW>();
        let total = 4 + item * self.offer.entries.len();

        unsafe {
            let handle = GlobalAlloc(GMEM_MOVEABLE, total)?;
            let base = GlobalLock(handle) as *mut u8;
            if base.is_null() {
                return Err(Error::from(E_NOTIMPL));
            }

            std::ptr::write_unaligned(base as *mut u32, self.offer.entries.len() as u32);
            for (index, entry) in self.offer.entries.iter().enumerate() {
                let descriptor = descriptor_for(entry);
                std::ptr::write_unaligned(
                    base.add(4 + index * item) as *mut FILEDESCRIPTORW,
                    descriptor,
                );
            }
            let _ = GlobalUnlock(handle);

            Ok(STGMEDIUM {
                tymed: TYMED_HGLOBAL.0 as u32,
                u: STGMEDIUM_0 { hGlobal: handle },
                pUnkForRelease: ManuallyDrop::new(None),
            })
        }
    }

    fn build_stream(&self, index: i32) -> WinResult<STGMEDIUM> {
        let entry = usize::try_from(index)
            .ok()
            .and_then(|i| self.offer.entries.get(i))
            .filter(|entry| !entry.is_dir)
            .ok_or_else(|| Error::from(DV_E_LINDEX))?;

        let stream: IStream = RemoteFileStream {
            offer: self.offer.clone(),
            index: index as u32,
            size: entry.size,
            cursor: Mutex::new(Cursor::default()),
        }
        .into();

        Ok(STGMEDIUM {
            tymed: TYMED_ISTREAM.0 as u32,
            u: STGMEDIUM_0 {
                pstm: ManuallyDrop::new(Some(stream)),
            },
            pUnkForRelease: ManuallyDrop::new(None),
        })
    }

    fn format_supported(&self, format: &FORMATETC) -> HRESULT {
        if format.dwAspect != DVASPECT_CONTENT.0 {
            return DV_E_FORMATETC;
        }

        if format.cfFormat == self.fmt_descriptor {
            return if format.tymed & TYMED_HGLOBAL.0 as u32 != 0 {
                S_OK
            } else {
                DV_E_TYMED
            };
        }

        if format.cfFormat == self.fmt_contents {
            return if format.tymed & TYMED_ISTREAM.0 as u32 != 0 {
                S_OK
            } else {
                DV_E_TYMED
            };
        }

        DV_E_FORMATETC
    }
}

fn descriptor_for(entry: &FileEntry) -> FILEDESCRIPTORW {
    let mut name = [0u16; 260];
    let path = entry.rel_path.replace('/', "\\");
    for (slot, unit) in name
        .iter_mut()
        .zip(path.encode_utf16().take(MAX_NAME_UNITS))
    {
        *slot = unit;
    }

    let mut flags = FD_ATTRIBUTES.0 as u32 | FD_PROGRESSUI.0 as u32;
    if !entry.is_dir {
        flags |= FD_FILESIZE.0 as u32;
    }

    FILEDESCRIPTORW {
        dwFlags: flags,
        clsid: Default::default(),
        sizel: Default::default(),
        pointl: Default::default(),
        dwFileAttributes: if entry.is_dir {
            FILE_ATTRIBUTE_DIRECTORY
        } else {
            FILE_ATTRIBUTE_NORMAL
        },
        ftCreationTime: FILETIME::default(),
        ftLastAccessTime: FILETIME::default(),
        ftLastWriteTime: FILETIME::default(),
        nFileSizeHigh: (entry.size >> 32) as u32,
        nFileSizeLow: entry.size as u32,
        cFileName: name,
    }
}

#[allow(non_snake_case)]
impl IDataObject_Impl for VirtualFiles {
    fn GetData(&self, pformatetcin: *const FORMATETC) -> WinResult<STGMEDIUM> {
        let format = unsafe { pformatetcin.as_ref() }.ok_or_else(|| Error::from(DV_E_FORMATETC))?;
        let supported = self.format_supported(format);
        if supported != S_OK {
            return Err(Error::from(supported));
        }

        if format.cfFormat == self.fmt_descriptor {
            self.build_descriptor()
        } else {
            self.build_stream(format.lindex)
        }
    }

    fn GetDataHere(&self, _: *const FORMATETC, _: *mut STGMEDIUM) -> WinResult<()> {
        Err(Error::from(E_NOTIMPL))
    }

    fn QueryGetData(&self, pformatetc: *const FORMATETC) -> HRESULT {
        match unsafe { pformatetc.as_ref() } {
            Some(format) => self.format_supported(format),
            None => DV_E_FORMATETC,
        }
    }

    fn GetCanonicalFormatEtc(&self, _: *const FORMATETC, _: *mut FORMATETC) -> HRESULT {
        DATA_S_SAMEFORMATETC
    }

    fn SetData(&self, _: *const FORMATETC, _: *const STGMEDIUM, _: BOOL) -> WinResult<()> {
        Err(Error::from(E_NOTIMPL))
    }

    fn EnumFormatEtc(&self, dwdirection: u32) -> WinResult<IEnumFORMATETC> {
        if dwdirection != DATADIR_GET.0 as u32 {
            return Err(Error::from(E_NOTIMPL));
        }

        let formats = [
            FORMATETC {
                cfFormat: self.fmt_descriptor,
                ptd: std::ptr::null_mut(),
                dwAspect: DVASPECT_CONTENT.0,
                lindex: -1,
                tymed: TYMED_HGLOBAL.0 as u32,
            },
            FORMATETC {
                cfFormat: self.fmt_contents,
                ptd: std::ptr::null_mut(),
                dwAspect: DVASPECT_CONTENT.0,
                lindex: -1,
                tymed: TYMED_ISTREAM.0 as u32,
            },
        ];
        unsafe { SHCreateStdEnumFmtEtc(&formats) }
    }

    fn DAdvise(&self, _: *const FORMATETC, _: u32, _: Option<&IAdviseSink>) -> WinResult<u32> {
        Err(Error::from(OLE_E_ADVISENOTSUPPORTED))
    }

    fn DUnadvise(&self, _: u32) -> WinResult<()> {
        Err(Error::from(OLE_E_ADVISENOTSUPPORTED))
    }

    fn EnumDAdvise(&self) -> WinResult<IEnumSTATDATA> {
        Err(Error::from(OLE_E_ADVISENOTSUPPORTED))
    }
}

/// 顺序读取时同时在途的块数；每块 [`CHUNK_SIZE`]，把「一块一往返」变成流水线以占满带宽。
const PREFETCH_WINDOW: usize = 8;

type ChunkTask = tauri::async_runtime::JoinHandle<anyhow::Result<Vec<u8>>>;

#[derive(Default)]
struct Cursor {
    position: u64,
    /// 最近一次拉取的块：`(起始偏移, 数据)`，让小块顺序读不必每次都往返网络。
    cache: Option<(u64, Vec<u8>)>,
    /// 已发出请求的后续块，按偏移递增排列。
    inflight: VecDeque<(u64, ChunkTask)>,
}

impl Cursor {
    fn abort_inflight(&mut self) {
        for (_, task) in self.inflight.drain(..) {
            task.abort();
        }
    }
}

impl Drop for Cursor {
    fn drop(&mut self) {
        self.abort_inflight();
    }
}

/// 按需向对端拉取内容的只读文件流。
#[implement(IStream)]
struct RemoteFileStream {
    offer: Arc<RemoteOffer>,
    index: u32,
    size: u64,
    cursor: Mutex<Cursor>,
}

impl RemoteFileStream {
    /// 把在途请求补满窗口：从 `from`（或最后一个在途块之后）起依次发出。
    fn top_up(&self, cursor: &mut Cursor, from: u64) {
        let mut next = cursor
            .inflight
            .back()
            .map_or(from, |(start, _)| *start + CHUNK_SIZE as u64);

        while cursor.inflight.len() < PREFETCH_WINDOW && next < self.size {
            let (state, peer_id, offer, index) = (
                self.offer.state.clone(),
                self.offer.peer_id.clone(),
                self.offer.offer.clone(),
                self.index,
            );
            let task = tauri::async_runtime::spawn(async move {
                fetch_file_range(&state, &peer_id, &offer, index, next, CHUNK_SIZE as u32).await
            });
            cursor.inflight.push_back((next, task));
            next += CHUNK_SIZE as u64;
        }
    }

    /// 从当前位置读取至多 `want` 字节；到达文件末尾返回空。
    fn read_at_cursor(&self, want: usize) -> Result<Vec<u8>, ()> {
        let mut cursor = self.cursor.lock().expect("stream cursor poisoned");
        if cursor.position >= self.size || want == 0 {
            return Ok(Vec::new());
        }

        let position = cursor.position;
        let covered = cursor.cache.as_ref().is_some_and(|(start, data)| {
            position >= *start && position < *start + data.len() as u64
        });

        if !covered {
            // 丢弃落在当前位置之前的在途块（发生了向前 seek 之类）。
            while cursor
                .inflight
                .front()
                .is_some_and(|(start, _)| *start < position)
            {
                if let Some((_, task)) = cursor.inflight.pop_front() {
                    task.abort();
                }
            }
            let hit = cursor
                .inflight
                .front()
                .is_some_and(|(start, _)| *start == position);
            if !hit {
                cursor.abort_inflight();
            }
            self.top_up(&mut cursor, position);

            let (_, task) = cursor.inflight.pop_front().expect("window is never empty");
            let data = tauri::async_runtime::block_on(task)
                .map_err(|err| log::warn!("sync: chunk task failed: {err}"))?
                .map_err(|err| {
                    log::warn!("sync: fetching remote file chunk failed: {err:#}");
                    cursor.abort_inflight();
                })?;

            if data.is_empty() {
                cursor.abort_inflight();
                return Ok(Vec::new());
            }
            let next = position + data.len() as u64;
            cursor.cache = Some((position, data));
            self.top_up(&mut cursor, next);
        }

        let (start, data) = cursor.cache.as_ref().expect("cache filled above");
        let from = (position - *start) as usize;
        let take = want.min(data.len() - from);
        let out = data[from..from + take].to_vec();
        cursor.position += take as u64;
        Ok(out)
    }
}

#[allow(non_snake_case)]
impl ISequentialStream_Impl for RemoteFileStream {
    fn Read(&self, pv: *mut c_void, cb: u32, pcbread: *mut u32) -> HRESULT {
        let mut filled = 0usize;
        let want = cb as usize;

        while filled < want {
            match self.read_at_cursor(want - filled) {
                Ok(chunk) if chunk.is_empty() => break,
                Ok(chunk) => unsafe {
                    std::ptr::copy_nonoverlapping(
                        chunk.as_ptr(),
                        (pv as *mut u8).add(filled),
                        chunk.len(),
                    );
                    filled += chunk.len();
                },
                Err(()) => {
                    if !pcbread.is_null() {
                        unsafe { *pcbread = filled as u32 };
                    }
                    return STG_E_READFAULT;
                }
            }
        }

        if !pcbread.is_null() {
            unsafe { *pcbread = filled as u32 };
        }
        S_OK
    }

    fn Write(&self, _: *const c_void, _: u32, _: *mut u32) -> HRESULT {
        STG_E_INVALIDFUNCTION
    }
}

#[allow(non_snake_case)]
impl IStream_Impl for RemoteFileStream {
    fn Seek(
        &self,
        dlibmove: i64,
        dworigin: STREAM_SEEK,
        plibnewposition: *mut u64,
    ) -> WinResult<()> {
        let mut cursor = self.cursor.lock().expect("stream cursor poisoned");
        let base = match dworigin {
            STREAM_SEEK_SET => 0i128,
            STREAM_SEEK_CUR => i128::from(cursor.position),
            STREAM_SEEK_END => i128::from(self.size),
            _ => return Err(Error::from(STG_E_INVALIDFUNCTION)),
        };

        let next = base + i128::from(dlibmove);
        if next < 0 {
            return Err(Error::from(STG_E_INVALIDFUNCTION));
        }

        cursor.position = next as u64;
        if !plibnewposition.is_null() {
            unsafe { *plibnewposition = cursor.position };
        }
        Ok(())
    }

    fn SetSize(&self, _: u64) -> WinResult<()> {
        Err(Error::from(E_NOTIMPL))
    }

    fn CopyTo(
        &self,
        pstm: Option<&IStream>,
        cb: u64,
        pcbread: *mut u64,
        pcbwritten: *mut u64,
    ) -> WinResult<()> {
        let target = pstm.ok_or_else(|| Error::from(STG_E_INVALIDFUNCTION))?;
        let mut remaining = cb;
        let (mut read_total, mut written_total) = (0u64, 0u64);

        while remaining > 0 {
            let want = remaining.min(CHUNK_SIZE as u64) as usize;
            let chunk = self
                .read_at_cursor(want)
                .map_err(|()| Error::from(STG_E_READFAULT))?;
            if chunk.is_empty() {
                break;
            }

            let mut written = 0u32;
            unsafe {
                target
                    .Write(
                        chunk.as_ptr().cast(),
                        chunk.len() as u32,
                        Some(&mut written),
                    )
                    .ok()?;
            }
            read_total += chunk.len() as u64;
            written_total += u64::from(written);
            remaining -= chunk.len() as u64;
        }

        unsafe {
            if !pcbread.is_null() {
                *pcbread = read_total;
            }
            if !pcbwritten.is_null() {
                *pcbwritten = written_total;
            }
        }
        Ok(())
    }

    fn Commit(&self, _: &STGC) -> WinResult<()> {
        Ok(())
    }

    fn Revert(&self) -> WinResult<()> {
        Ok(())
    }

    fn LockRegion(&self, _: u64, _: u64, _: &LOCKTYPE) -> WinResult<()> {
        Err(Error::from(STG_E_INVALIDFUNCTION))
    }

    fn UnlockRegion(&self, _: u64, _: u64, _: u32) -> WinResult<()> {
        Err(Error::from(STG_E_INVALIDFUNCTION))
    }

    fn Stat(&self, pstatstg: *mut STATSTG, _: &STATFLAG) -> WinResult<()> {
        if pstatstg.is_null() {
            return Err(Error::from(STG_E_INVALIDFUNCTION));
        }

        unsafe {
            *pstatstg = STATSTG {
                pwcsName: windows::core::PWSTR::null(),
                r#type: STGTY_STREAM.0 as u32,
                cbSize: self.size,
                mtime: FILETIME::default(),
                ctime: FILETIME::default(),
                atime: FILETIME::default(),
                grfMode: STGM(0),
                grfLocksSupported: 0,
                clsid: Default::default(),
                grfStateBits: 0,
                reserved: 0,
            };
        }
        Ok(())
    }

    fn Clone(&self) -> WinResult<IStream> {
        let (position, cache) = {
            let guard = self.cursor.lock().expect("stream cursor poisoned");
            (guard.position, guard.cache.clone())
        };
        let cursor = Cursor {
            position,
            cache,
            inflight: VecDeque::new(),
        };
        Ok(RemoteFileStream {
            offer: self.offer.clone(),
            index: self.index,
            size: self.size,
            cursor: Mutex::new(cursor),
        }
        .into())
    }
}
