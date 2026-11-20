package settings

import (
	"encoding/json"
	"slices"
)

// UnmarshalJSON drops retired actions in old settings and backup patches
// before validation, preserving active selections and their order.
func (c *Content) UnmarshalJSON(data []byte) error {
	type content Content
	next := content(*c)
	if err := json.Unmarshal(data, &next); err != nil {
		return err
	}
	for _, actions := range []*[]ItemAction{&next.ItemActions, &next.ItemActionOrder} {
		*actions = slices.DeleteFunc(slices.Clone(*actions), func(action ItemAction) bool {
			return action == "pastePath" || action == "pastePlain"
		})
	}
	*c = Content(next)
	return nil
}
