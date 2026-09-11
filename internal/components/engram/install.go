package engram

import (
	"github.com/pablogore/atomwright/v2/internal/installcmd"
	"github.com/pablogore/atomwright/v2/internal/model"
	"github.com/pablogore/atomwright/v2/internal/system"
)

func InstallCommand(profile system.PlatformProfile) ([][]string, error) {
	return installcmd.NewResolver().ResolveComponentInstall(profile, model.ComponentEngram)
}
