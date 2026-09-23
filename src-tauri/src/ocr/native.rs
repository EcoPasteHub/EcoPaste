use std::path::Path;

const MAX_INPUT_BYTES: u64 = 20 * 1024 * 1024;
const MAX_PIXELS: u64 = 40_000_000;
const MAX_OUTPUT_CHARS: usize = 100_000;

#[cfg(target_os = "windows")]
#[path = "windows.rs"]
mod windows;

#[cfg(target_os = "windows")]
pub use windows::{configure, recognize, supported};

#[cfg(target_os = "macos")]
pub fn supported() -> bool {
    true
}

/// Apple Vision needs no bundled model; Windows validates resources before reporting readiness.
#[cfg(target_os = "macos")]
pub fn configure(_model_dir: &Path) -> anyhow::Result<()> {
    Ok(())
}

/// Reads a single bounded snapshot and validates dimensions before either native engine sees it.
fn read_image(path: &Path) -> anyhow::Result<(Vec<u8>, u32, u32)> {
    use anyhow::{ensure, Context};
    use std::io::{Cursor, Read};

    let file = std::fs::File::open(path).context("Cannot open OCR image")?;
    let metadata = file.metadata().context("Cannot inspect OCR image")?;
    ensure!(metadata.is_file(), "OCR image must be a regular file");
    ensure!(
        metadata.len() <= MAX_INPUT_BYTES,
        "OCR image exceeds 20 MiB"
    );
    let mut bytes = Vec::new();
    file.take(MAX_INPUT_BYTES + 1)
        .read_to_end(&mut bytes)
        .context("Cannot read OCR image")?;
    ensure!(
        bytes.len() as u64 <= MAX_INPUT_BYTES,
        "OCR image exceeds 20 MiB"
    );
    let (width, height) = image::ImageReader::new(Cursor::new(&bytes))
        .with_guessed_format()
        .context("Invalid OCR image format")?
        .into_dimensions()
        .context("Invalid OCR image dimensions")?;
    ensure!(width > 0 && height > 0, "OCR image is empty");
    ensure!(
        u64::from(width) * u64::from(height) <= MAX_PIXELS,
        "OCR image exceeds 40 megapixels"
    );
    Ok((bytes, width, height))
}

/// Runs local Vision recognition synchronously; callers must use a blocking worker.
/// The immutable, bounded byte snapshot is also the image validated below.
#[cfg(target_os = "macos")]
pub fn recognize(path: &Path) -> anyhow::Result<String> {
    use anyhow::Context;
    use objc2::{rc::autoreleasepool, AnyThread};
    use objc2_foundation::{NSArray, NSData, NSDictionary, NSString};
    use objc2_vision::{
        VNImageRequestHandler, VNRecognizeTextRequest, VNRequest, VNRequestTextRecognitionLevel,
    };
    let (bytes, _, _) = read_image(path)?;

    autoreleasepool(|_| {
        let request = VNRecognizeTextRequest::new();
        request.setRecognitionLevel(VNRequestTextRecognitionLevel::Accurate);
        request.setRecognitionLanguages(&NSArray::from_retained_slice(&[
            NSString::from_str("zh-Hans"),
            NSString::from_str("en-US"),
        ]));
        request.setUsesLanguageCorrection(true);

        let data = NSData::from_vec(bytes);
        let handler = VNImageRequestHandler::initWithData_options(
            VNImageRequestHandler::alloc(),
            &data,
            &NSDictionary::new(),
        );
        let requests = NSArray::<VNRequest>::from_slice(&[&request]);
        handler.performRequests_error(&requests).map_err(|error| {
            anyhow::anyhow!(
                "Apple Vision could not recognize this image (error {})",
                error.code()
            )
        })?;

        let observations = request
            .results()
            .context("Apple Vision returned no result")?;
        Ok(bounded_text(observations.iter().filter_map(
            |observation| {
                observation
                    .topCandidates(1)
                    .firstObject()
                    .map(|text| text.string().to_string())
            },
        )))
    })
}

/// Joins nonempty observations without splitting UTF-8 or exceeding the index limit.
fn bounded_text<S: AsRef<str>>(lines: impl IntoIterator<Item = S>) -> String {
    let mut output = String::new();
    let mut remaining = MAX_OUTPUT_CHARS;
    for line in lines {
        let line = line.as_ref().trim();
        if line.is_empty() {
            continue;
        }
        if !output.is_empty() {
            output.push('\n');
            remaining -= 1;
        }
        for character in line.chars().take(remaining) {
            output.push(character);
            remaining -= 1;
        }
        if remaining == 0 {
            break;
        }
    }
    output
}

#[cfg(test)]
mod tests {
    use super::*;

    #[cfg(target_os = "macos")]
    fn fixture(name: &str) -> std::path::PathBuf {
        Path::new(env!("CARGO_MANIFEST_DIR"))
            .join("src/ocr/native_fixtures")
            .join(name)
    }

    #[test]
    #[cfg(target_os = "macos")]
    fn recognizes_generated_english_image() {
        assert!(supported());
        let result = recognize(&fixture("english.png")).unwrap();
        assert!(result.contains("EcoPaste local image search"));
    }

    #[test]
    #[cfg(target_os = "macos")]
    fn recognizes_generated_chinese_image() {
        let result = recognize(&fixture("chinese.png")).unwrap();
        assert!(result.contains("本地图片文字识别测试"));
    }

    #[test]
    fn rejects_missing_file() {
        let dir = tempfile::tempdir().unwrap();
        let error = read_image(&dir.path().join("missing.png")).unwrap_err();
        assert!(error.to_string().contains("open"));
    }

    #[test]
    fn rejects_invalid_image() {
        let file = tempfile::NamedTempFile::new().unwrap();
        std::fs::write(file.path(), b"this is not an image").unwrap();
        let error = read_image(file.path()).unwrap_err();
        assert!(error.to_string().contains("image"));
    }

    #[test]
    fn rejects_file_above_twenty_mib() {
        let file = tempfile::NamedTempFile::new().unwrap();
        file.as_file().set_len(20 * 1024 * 1024 + 1).unwrap();
        let error = read_image(file.path()).unwrap_err();
        assert!(error.to_string().contains("20 MiB"));
    }

    #[test]
    fn rejects_image_above_forty_megapixels() {
        let file = tempfile::NamedTempFile::new().unwrap();
        image::GrayImage::new(6400, 6400)
            .save_with_format(file.path(), image::ImageFormat::Png)
            .unwrap();
        let error = read_image(file.path()).unwrap_err();
        assert!(error.to_string().contains("40 megapixels"));
    }

    #[test]
    #[cfg(target_os = "macos")]
    fn blank_image_has_no_text() {
        let file = tempfile::NamedTempFile::new().unwrap();
        image::GrayImage::from_pixel(640, 480, image::Luma([255]))
            .save_with_format(file.path(), image::ImageFormat::Png)
            .unwrap();
        assert!(recognize(file.path()).unwrap().is_empty());
    }

    #[test]
    fn output_limit_counts_unicode_characters_and_separators() {
        let result = bounded_text(["文".repeat(99_998), "😀文".into()]);
        assert_eq!(result.chars().count(), 100_000);
        assert!(result.ends_with("\n😀"));
    }
}
