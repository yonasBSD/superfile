package internal

import (
	"github.com/yorukot/superfile/src/internal/common"
	"github.com/yorukot/superfile/src/internal/common/utils"
)

// Todo : Move their usage to direct access, instead of alias
// We have made alias to temporarily avoid more changes for now
var Config = common.Config
var theme = common.Theme
var LogAndExit = utils.LogAndExit