package minisocial

import (
	"std"  // The standard Gno package
	"time" // For handling time operations
)

// Post defines the main data we keep about each post
type Post struct {
	text      string      // Main text body
	author    std.Address // Address of the post author, provided by the execution context
	createdAt time.Time   // When the post was created
}
