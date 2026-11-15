package ui

import (
	"context"

	"github.com/EcoPasteHub/EcoPaste/internal/native/macos"
)

var historyRowEvents = map[string]func(*historyPanel, any){
	"toggleFavorite": toggleRowFavorite,
	"togglePinned":   toggleRowPinned,
	"delete":         deleteRow,
	"preview":        previewRow,
	"hidePreview":    hideRowPreview,
	"open":           openRow,
	"reveal":         revealRow,
	"saveImage":      saveRowImage,
	"editNote":       editRowNote,
	"moveGroup":      moveRowGroup,
}

var historyGroupEvents = map[string]func(*historyPanel, any){
	"newGroup":     createGroup,
	"renameGroup":  renameGroup,
	"deleteGroup":  deleteGroup,
	"clearHistory": clearHistory,
}

func toggleRowFavorite(p *historyPanel, data any) {
	p.actions.run(p.actionID(data), true, func(ctx context.Context, id string) error {
		_, err := p.model.ToggleFavorite(ctx, id)
		return err
	})
}

func toggleRowPinned(p *historyPanel, data any) {
	p.actions.run(p.actionID(data), true, func(ctx context.Context, id string) error {
		_, err := p.model.TogglePinned(ctx, id)
		return err
	})
}

func deleteRow(p *historyPanel, data any) {
	id := p.actionID(data)
	if id == "" || p.model == nil {
		return
	}
	ctx := p.cx.Context()
	go func() {
		err := p.model.Delete(ctx, id)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			if p.selectedID == id {
				p.selectedID = ""
			}
			p.reload()
		})
	}()
}

func previewRow(p *historyPanel, _ any) {
	if p.preview.GetUntracked().Active {
		p.preview.Set(macos.PreviewContent{})
		return
	}
	p.actions.showPreview(p.selectedID)
}

func hideRowPreview(p *historyPanel, _ any) {
	p.preview.Set(macos.PreviewContent{})
}

func openRow(p *historyPanel, data any) {
	if p.model != nil {
		p.actions.run(p.actionID(data), false, p.model.Open)
	}
}

func revealRow(p *historyPanel, data any) {
	if p.model != nil {
		p.actions.run(p.actionID(data), false, p.model.Reveal)
	}
}

func saveRowImage(p *historyPanel, data any) {
	if p.model != nil {
		p.actions.run(p.actionID(data), false, p.model.SaveImage)
	}
}

func editRowNote(p *historyPanel, data any) {
	id := p.actionID(data)
	if id == "" || p.model == nil {
		return
	}
	ctx := p.cx.Context()
	go func() {
		saved, err := p.model.EditNote(ctx, id)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			if saved {
				p.reload()
			}
		})
	}()
}

func moveRowGroup(p *historyPanel, data any) {
	groupID, _ := data.(string)
	p.actions.run(p.selectedID, true, func(ctx context.Context, id string) error {
		return p.model.MoveToGroup(ctx, id, groupID)
	})
}

func createGroup(p *historyPanel, _ any) {
	if p.model == nil {
		return
	}
	ctx := p.cx.Context()
	go func() {
		created, ok, err := p.model.CreateGroup(ctx)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			if ok && created != nil {
				p.loadGroups(created.ID, true)
			}
		})
	}()
}

func renameGroup(p *historyPanel, _ any) {
	if p.model == nil || p.groupID == "" {
		return
	}
	ctx := p.cx.Context()
	go func(id string) {
		renamed, err := p.model.RenameGroup(ctx, id)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			if renamed {
				p.loadGroups(id, false)
			}
		})
	}(p.groupID)
}

func deleteGroup(p *historyPanel, _ any) {
	if p.model == nil || p.groupID == "" {
		return
	}
	ctx := p.cx.Context()
	go func(id string) {
		deleted, err := p.model.DeleteGroup(ctx, id)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			if deleted {
				p.groupID = ""
				p.group.Set("")
				p.loadGroups("", true)
			}
		})
	}(p.groupID)
}

func clearHistory(p *historyPanel, _ any) {
	if p.model == nil {
		return
	}
	ctx := p.cx.Context()
	go func() {
		err := p.model.ClearHistory(ctx)
		p.cx.Post(func() {
			if err != nil {
				p.errorText.Set(err.Error())
				return
			}
			p.selectedID = ""
			p.reload()
		})
	}()
}
