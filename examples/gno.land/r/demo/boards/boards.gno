package boards

import (
	"regexp"
	"std"

	"gno.land/p/demo/avl"
	"gno.land/p/moul/txlink"
)

//----------------------------------------
// Realm (package) state

var (
	gRealmLink      txlink.Realm
	gBoards         avl.Tree    // id -> *Board
	gBoardsCtr      int         // increments Board.id
	gBoardsByName   avl.Tree    // name -> *Board
	gDefaultAnonFee = 100000000 // minimum fee required if anonymous
)

//----------------------------------------
// Constants

var reName = regexp.MustCompile(`^[a-z]+[_a-z0-9]{2,29}$`)

func init() {
	// Keep track of the realm package path to make it available on render
	gRealmLink = txlink.Realm(std.CurrentRealm().PkgPath())
}
