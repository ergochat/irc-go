package eventmgr

import "testing"

var test2Tracker = 0

func handler1(event string, info InfoMap) {
	t := info["test"].(*testing.T)

	if info["k"].(int) != 3 {
		t.Error(
			"For", event,
			"expected", 3,
			"got", info["k"],
		)
	}
}

func handler2First(event string, info InfoMap) {
	t := info["test"].(*testing.T)

	if test2Tracker != 0 {
		t.Error(
			"For", event,
			"expected", 0,
			"got", test2Tracker,
		)
	} else {
		test2Tracker += 3
	}
}

func handler2Second(event string, info InfoMap) {
	t := info["test"].(*testing.T)

	if test2Tracker != 3 {
		t.Error(
			"For", event,
			"expected", 3,
			"got", test2Tracker,
		)
	} else {
		test2Tracker--
	}
}

func handler2Third(event string, info InfoMap) {
	t := info["test"].(*testing.T)

	if test2Tracker != 2 {
		t.Error(
			"For", event,
			"expected", 2,
			"got", test2Tracker,
		)
	} else {
		test2Tracker += 5
	}
}

func TestAttachDispatch(t *testing.T) {
	var manager EventManager

	// test dispatching
	manager.Attach("test1", handler1, 0)
	info := NewInfoMap()
	info["k"] = 3
	info["test"] = t
	manager.Dispatch("test1", info)

	// test priority dispatching
	manager.Attach("test2", handler2Second, 8)
	manager.Attach("test2", handler2First, 3)
	manager.Attach("test2", handler2Third, 16)
	info = make(InfoMap)
	info["test"] = t
	manager.Dispatch("test2", info)

	if test2Tracker != 7 {
		t.Error(
			"For", "test2 end",
			"expected", 7,
			"got", test2Tracker,
		)
	}
}
