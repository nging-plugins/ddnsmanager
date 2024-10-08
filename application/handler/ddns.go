package handler

import (
	"context"

	"github.com/webx-top/echo"
	"github.com/webx-top/echo/code"

	"github.com/coscms/webcore/library/common"
	"github.com/coscms/webcore/library/webhook"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/boot"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/config"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/domain/dnsdomain"
	_ "github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/providerall"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/utils"
)

func DdnsSettings(ctx echo.Context) error {
	var err error
	cfg := boot.Config()
	if ctx.IsPost() {
		if ctx.Formx(`force`).Bool() {
			err = boot.ForceUpdate(ctx)
			if err != nil {
				return ctx.JSON(ctx.Data().SetError(err))
			}
			return ctx.JSON(ctx.Data().SetInfo(ctx.T(`执行成功`), code.Success.Int()))
		}
		cfg = config.New()
		if err = ctx.MustBindAndValidate(cfg); err != nil {
			goto END
		}
		//echo.Dump(cfg)
		err = boot.SetConfig(cfg)
		if err != nil {
			goto END
		}
		err = boot.Reset(context.Background())
		if err != nil {
			goto END
		}
		common.SendOk(ctx, ctx.T(`保存成功`))
		ctx.Set(`isRunning`, boot.IsRunning())
		return ctx.Redirect(`/tool/ddns`)
	}

END:
	ctx.Set(`config`, cfg)
	ctx.Set(`ttlList`, config.TTLs.Slice())
	ctx.Set(`providers`, ddnsmanager.AllProvoderMeta(cfg.DNSServices))
	ctx.Set(`title`, `DDNS`)
	ipv4NetInterfaces, ipv6NetInterfaces, _ := utils.GetNetInterface(``)
	ctx.Set(`ipv4NetInterfaces`, ipv4NetInterfaces)
	ctx.Set(`ipv6NetInterfaces`, ipv6NetInterfaces)
	ctx.Set(`tagValueDescs`, dnsdomain.TagValueDescs.Slice())
	ctx.Set(`trackerTypes`, dnsdomain.TrackerTypes.Slice())
	ctx.Set(`httpMethods`, webhook.Methods)
	ctx.Set(`isRunning`, boot.IsRunning())
	return ctx.Render(`ddns/ddns`, common.Err(ctx, err))
}
