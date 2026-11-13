//go:build darwin

package macos

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	eventClassKeyboard   = 0x6B657962
	eventHotKeyPressed   = 5
	eventParamDirectObj  = 0x2D2D2D2D
	typeEventHotKeyID    = 0x686B6964
	eventHotKeySignature = 0x45636F50
	hotKeyExistsErr      = -9878

	carbonCmd     = 1 << 8
	carbonShift   = 1 << 9
	carbonOption  = 1 << 11
	carbonControl = 1 << 12
)

type eventTypeSpec struct {
	class uint32
	kind  uint32
}

type eventHotKeyID struct {
	signature uint32
	id        uint32
}

var (
	carbonHotKeyOnce sync.Once
	carbonHotKeyErr  error

	getApplicationEventTarget func() uintptr
	registerEventHotKeyFunc   func(uint32, uint32, eventHotKeyID, uintptr, uint32, *uint32) int32
	unregisterEventHotKeyFunc func(uint32) int32
	installEventHandlerFunc   func(uintptr, uintptr, uint64, *eventTypeSpec, uintptr, *uintptr) int32
	getEventParameterFunc     func(uintptr, uint32, uint32, uint32, uint64, *uint64, unsafe.Pointer) int32

	carbonHotKeyMu        sync.Mutex
	carbonHotKeyCallbacks = map[uint32]func(){}
)

// HotKeyBinding is one global shortcut and the action it runs on the main
// thread.
type HotKeyBinding struct {
	Key    string
	Action func()
}

// HotKeyManager owns Carbon global hotkeys, which can be replaced at
// runtime without restarting the app.
type HotKeyManager struct {
	mu   sync.Mutex
	refs []uint32
	ids  []uint32
	next uint32
}

// NewHotKeyManager installs the Carbon event handler once.
func NewHotKeyManager() *HotKeyManager {
	return &HotKeyManager{}
}

// Set replaces every registered shortcut with the given bindings.
func (m *HotKeyManager) Set(bindings []HotKeyBinding) error {
	loadCarbonHotKeys()
	if carbonHotKeyErr != nil {
		return carbonHotKeyErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var result error
	onMain(func() {
		for i, ref := range m.refs {
			unregisterEventHotKeyFunc(ref)
			carbonHotKeyMu.Lock()
			delete(carbonHotKeyCallbacks, m.ids[i])
			carbonHotKeyMu.Unlock()
		}
		m.refs = nil
		m.ids = nil
		for _, binding := range bindings {
			code, modifiers, ok := parseHotKey(binding.Key)
			if !ok || binding.Action == nil {
				result = fmt.Errorf("无效的快捷键: %s", binding.Key)
				continue
			}
			m.next++
			id := m.next
			var ref uint32
			status := registerEventHotKeyFunc(code, modifiers,
				eventHotKeyID{signature: eventHotKeySignature, id: id},
				getApplicationEventTarget(), 0, &ref)
			if status != 0 {
				if status == hotKeyExistsErr {
					result = fmt.Errorf("快捷键 %s 已被其他应用占用", binding.Key)
				} else {
					result = fmt.Errorf("无法注册快捷键 %s", binding.Key)
				}
				continue
			}
			carbonHotKeyMu.Lock()
			carbonHotKeyCallbacks[id] = binding.Action
			carbonHotKeyMu.Unlock()
			m.refs = append(m.refs, ref)
			m.ids = append(m.ids, id)
		}
	})
	return result
}

// Close unregisters every shortcut this manager owns.
func (m *HotKeyManager) Close() {
	_ = m.Set(nil)
}

func loadCarbonHotKeys() {
	carbonHotKeyOnce.Do(func() {
		carbon, err := purego.Dlopen("/System/Library/Frameworks/Carbon.framework/Carbon",
			purego.RTLD_GLOBAL|purego.RTLD_NOW)
		if err != nil {
			carbonHotKeyErr = fmt.Errorf("加载 Carbon: %w", err)
			return
		}
		purego.RegisterLibFunc(&getApplicationEventTarget, carbon, "GetApplicationEventTarget")
		purego.RegisterLibFunc(&registerEventHotKeyFunc, carbon, "RegisterEventHotKey")
		purego.RegisterLibFunc(&unregisterEventHotKeyFunc, carbon, "UnregisterEventHotKey")
		purego.RegisterLibFunc(&installEventHandlerFunc, carbon, "InstallEventHandler")
		purego.RegisterLibFunc(&getEventParameterFunc, carbon, "GetEventParameter")

		spec := eventTypeSpec{class: eventClassKeyboard, kind: eventHotKeyPressed}
		handler := purego.NewCallback(func(_, event, _ uintptr) int32 {
			var id eventHotKeyID
			size := uint64(unsafe.Sizeof(id))
			if getEventParameterFunc(event, eventParamDirectObj, typeEventHotKeyID, 0,
				size, &size, unsafe.Pointer(&id)) != 0 {
				return 0
			}
			carbonHotKeyMu.Lock()
			action := carbonHotKeyCallbacks[id.id]
			carbonHotKeyMu.Unlock()
			if action != nil {
				action()
			}
			return 0
		})
		var handlerRef uintptr
		if status := installEventHandlerFunc(getApplicationEventTarget(), handler, 1,
			&spec, 0, &handlerRef); status != 0 {
			carbonHotKeyErr = fmt.Errorf("安装全局快捷键处理器失败")
		}
	})
}

func parseHotKey(value string) (uint32, uint32, bool) {
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(value, " ", "")), "+")
	if len(parts) == 0 || parts[len(parts)-1] == "" {
		return 0, 0, false
	}
	key := parts[len(parts)-1]
	switch key {
	case "arrowup":
		key = "up"
	case "arrowdown":
		key = "down"
	case "arrowleft":
		key = "left"
	case "arrowright":
		key = "right"
	case "esc":
		key = "escape"
	case "return":
		key = "enter"
	}
	code, ok := hotKeyVirtualKeys[key]
	if !ok {
		return 0, 0, false
	}
	var modifiers uint32
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "command", "cmd", "meta", "mod":
			modifiers |= carbonCmd
		case "control", "ctrl":
			modifiers |= carbonControl
		case "alt", "option":
			modifiers |= carbonOption
		case "shift":
			modifiers |= carbonShift
		default:
			return 0, 0, false
		}
	}
	return uint32(code), modifiers, true
}

var hotKeyVirtualKeys = map[string]uint16{
	"a": 0, "s": 1, "d": 2, "f": 3, "h": 4, "g": 5, "z": 6, "x": 7, "c": 8, "v": 9,
	"b": 11, "q": 12, "w": 13, "e": 14, "r": 15, "y": 16, "t": 17, "1": 18, "2": 19,
	"3": 20, "4": 21, "6": 22, "5": 23, "=": 24, "plus": 24, "9": 25, "7": 26,
	"-": 27, "minus": 27, "8": 28, "0": 29, "]": 30, "o": 31, "u": 32, "[": 33,
	"i": 34, "p": 35, "enter": 36, "l": 37, "j": 38, "'": 39, "k": 40, ";": 41,
	"\\": 42, ",": 43, "/": 44, "n": 45, "m": 46, ".": 47, "tab": 48, "space": 49,
	"`": 50, "backspace": 51, "escape": 53, "f17": 64, "f18": 79, "f19": 80,
	"f20": 90, "f5": 96, "f6": 97, "f7": 98, "f3": 99, "f8": 100, "f9": 101,
	"f11": 103, "f13": 105, "f16": 106, "f14": 107, "f10": 109, "f12": 111,
	"f15": 113, "home": 115, "pageup": 116, "delete": 117, "f4": 118, "end": 119,
	"f2": 120, "pagedown": 121, "f1": 122, "left": 123, "right": 124, "down": 125, "up": 126,
}
