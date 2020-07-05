package runtime

import "github.com/gobuffalo/packr"

func New() packr.Box {
	return packr.NewBox("../../runtime")
}
