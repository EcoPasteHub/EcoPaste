//go:build !darwin

package pasteboard

import (
	"errors"

	"github.com/EcoPasteHub/EcoPaste/internal/clipboard"
	"github.com/EcoPasteHub/EcoPaste/internal/settings"
)

type Pasteboard struct{}

type App struct {
	BundleID string
	Name     string
}

func General() (*Pasteboard, error)      { return nil, errors.New("macOS only") }
func Named(string) (*Pasteboard, error)  { return nil, errors.New("macOS only") }
func (p *Pasteboard) ChangeCount() int64 { return 0 }
func (p *Pasteboard) Read(settings.Capture) (clipboard.Payload, bool, error) {
	return clipboard.Payload{}, false, errors.New("macOS only")
}
func (p *Pasteboard) WriteText(string, string, string, bool) {}
func (p *Pasteboard) WriteImage([]byte) error                { return errors.New("macOS only") }
func (p *Pasteboard) WriteFiles([]string) error              { return errors.New("macOS only") }
func Frontmost() App                                         { return App{} }
