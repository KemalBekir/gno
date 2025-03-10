// Package hof is the hall of fame realm.
// The Hall of Fame is an exhibition that holds items. Users can add their realms to the Hall of Fame by
// importing the Hall of Fame realm and calling hof.Register() from their init function.
package hof

import (
	"std"

	"gno.land/p/demo/avl"
	"gno.land/p/demo/ownable"
	"gno.land/p/demo/pausable"
	"gno.land/p/demo/seqid"

	"gno.land/r/leon/config"
)

var (
	exhibition *Exhibition

	// Safe objects
	Ownable  *ownable.Ownable
	Pausable *pausable.Pausable
)

type (
	Exhibition struct {
		itemCounter seqid.ID
		description string
		items       *avl.Tree // pkgPath > &Item
		itemsSorted *avl.Tree // same data but sorted, storing pointers
	}

	Item struct {
		id       seqid.ID
		pkgpath  string
		blockNum int64
		upvote   *avl.Tree // std.Addr > struct{}{}
		downvote *avl.Tree // std.Addr > struct{}{}
	}
)

func init() {
	exhibition = &Exhibition{
		items:       avl.NewTree(),
		itemsSorted: avl.NewTree(),
	}

	Ownable = ownable.NewWithAddress(config.OwnableMain.Owner()) // OrigSendOwnable?
	Pausable = pausable.NewFromOwnable(Ownable)
}

// Register registers your realm to the Hall of Fame
// Should be called from within code
func Register() {
	if Pausable.IsPaused() {
		return
	}

	submission := std.PreviousRealm()
	pkgpath := submission.PkgPath()

	// Must be called from code
	if submission.IsUser() {
		return
	}

	// Must not yet exist
	if exhibition.items.Has(pkgpath) {
		return
	}

	id := exhibition.itemCounter.Next()
	i := &Item{
		id:       id,
		pkgpath:  pkgpath,
		blockNum: std.ChainHeight(),
		upvote:   avl.NewTree(),
		downvote: avl.NewTree(),
	}

	exhibition.items.Set(pkgpath, i)
	exhibition.itemsSorted.Set(id.String(), i)

	std.Emit("Registration")
}

func Upvote(pkgpath string) {
	rawItem, ok := exhibition.items.Get(pkgpath)
	if !ok {
		panic(ErrNoSuchItem)
	}

	item := rawItem.(*Item)
	caller := std.PreviousRealm().Address().String()

	if item.upvote.Has(caller) {
		panic(ErrDoubleUpvote)
	}

	item.upvote.Set(caller, struct{}{})
}

func Downvote(pkgpath string) {
	rawItem, ok := exhibition.items.Get(pkgpath)
	if !ok {
		panic(ErrNoSuchItem)
	}

	item := rawItem.(*Item)
	caller := std.PreviousRealm().Address().String()

	if item.downvote.Has(caller) {
		panic(ErrDoubleDownvote)
	}

	item.downvote.Set(caller, struct{}{})
}

func Delete(pkgpath string) {
	if !Ownable.CallerIsOwner() {
		panic(ownable.ErrUnauthorized)
	}

	i, ok := exhibition.items.Get(pkgpath)
	if !ok {
		panic(ErrNoSuchItem)
	}

	if _, removed := exhibition.itemsSorted.Remove(i.(*Item).id.String()); !removed {
		panic(ErrNoSuchItem)
	}

	if _, removed := exhibition.items.Remove(pkgpath); !removed {
		panic(ErrNoSuchItem)
	}
}
