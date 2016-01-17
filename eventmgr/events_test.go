package eventmgr

import "testing"

func handler1(event string, info infomap) {
	t := info["test"].(*testing.T)

	if info["k"].(int) != 3 {
		t.Error(
			"For", event,
			"expected", 3,
			"got", info["k"],
		)
	}
}

func TestEncode(t *testing.T) {
	var manager EventManager

	manager.Attach("test1", handler1, 0)
	info := make(infomap)
	info["k"] = 3
	info["test"] = t
	manager.Dispatch("test1", info)
}
