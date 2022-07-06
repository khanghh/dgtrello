package discordbot

import (
	"github.com/lus/dgc"
)

func restrictRolesMiddleware(next dgc.ExecutionHandler) dgc.ExecutionHandler {
	return func(ctx *dgc.Ctx) {
		for _, allowedRoleId := range ctx.Command.Flags {
			for _, roleId := range ctx.Event.Member.Roles {
				if roleId == allowedRoleId {
					next(ctx)
					return
				}
			}
		}
		ctx.RespondText("You do not have permission to perform this action.")
	}
}
