//go:build darwin

package macos

import (
	"github.com/ebitengine/purego/objc"
)

func historyRowAtEvent(table, event objc.ID) int {
	location := objc.Send[CGPoint](event, sel("locationInWindow"))
	local := objc.Send[CGPoint](table, sel("convertPoint:fromView:"), location, objc.ID(0))
	return int(objc.Send[int64](table, sel("rowAtPoint:"), local))
}

func registerHistoryTargets() {
	var err error
	historyTableClass, err = objc.RegisterClass("EcoPasteHistoryTable", class("NSTableView"), nil, nil, historyTableMethods())
	if err != nil {
		panic(err)
	}
	if err := registerHistoryRowViews(); err != nil {
		panic(err)
	}
	target, err := objc.RegisterClass("EcoPasteHistoryTarget", class("NSObject"), nil, nil, historyTargetMethods())
	if err != nil {
		panic(err)
	}
	historyTarget = objc.ID(target).Send(sel("new"))
}
