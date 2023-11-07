package domain

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/admpub/log"
	"github.com/webx-top/echo"

	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/config"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/domain/dnsdomain"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/resolver"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/sender"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsmanager/utils"
	"github.com/nging-plugins/ddnsmanager/application/library/ddnsretry"
)

func (domains *Domains) SetIPv4Addr(ipv4Addr string) {
	domains.IPv4Addr = ipv4Addr
	domains.SaveIP(4)
}

func (domains *Domains) updateIPv4One(ctx context.Context, conf *config.Config, ipv4Addr string, dnsProvider string, wg *sync.WaitGroup, chanErrors chan error) error {
	dnsDomains := domains.IPv4Domains[dnsProvider]
	var _dnsDomains []*dnsdomain.Domain
	for _, dnsDomain := range dnsDomains {
		if dnsDomain == nil {
			continue
		}
		oldIP, err := resolver.ResolveDNS(dnsDomain.String(), conf.DNSResolver, `IPV4`)
		if err != nil {
			log.Errorf("[%s] ResolveDNS(%s): %s", dnsProvider, dnsDomain.String(), err.Error())
			//errs = append(errs, err)
			dnsDomain.UpdateStatus = dnsdomain.UpdatedIdle
			_dnsDomains = append(_dnsDomains, dnsDomain)
			continue
		}
		if oldIP != ipv4Addr {
			dnsDomain.UpdateStatus = dnsdomain.UpdatedIdle
			_dnsDomains = append(_dnsDomains, dnsDomain)
			continue
		}
		dnsDomain.UpdateStatus = dnsdomain.UpdatedNothing
		log.Infof("[%s] IP is the same as cached one (%s). Skip update (%s)", dnsProvider, ipv4Addr, dnsDomain.String())
	}
	if len(_dnsDomains) == 0 {
		return echo.ErrContinue
	}
	updater := ddnsmanager.Open(dnsProvider)
	if updater == nil {
		return echo.ErrContinue
	}
	dnsService := conf.FindService(dnsProvider)
	err := updater.Init(dnsService.Settings, _dnsDomains)
	if err != nil {
		chanErrors <- err
		return echo.ErrContinue
	}
	log.Infof("[%s] %s - Start to update record IP...", dnsProvider, ipv4Addr)
	err = updater.Update(ctx, `A`, ipv4Addr)
	if err != nil {
		log.Errorf("[%s] %s - Failed to update IP: %v (Wait to try again later)", dnsProvider, ipv4Addr, err)
		wg.Add(1)
		go func() {
			defer wg.Done()
			chanErrors <- ddnsretry.Retry(ctx, func(retryCtx context.Context) error {
				err := updater.Update(retryCtx, `A`, ipv4Addr)
				if err != nil {
					err = fmt.Errorf("[%s] %s - Failed to update IP: %v", dnsProvider, ipv4Addr, err)
				}
				return err
			})
		}()
		return echo.ErrContinue
	}
	return err
}

func (domains *Domains) SetIPv6Addr(ipv6Addr string) {
	domains.IPv6Addr = ipv6Addr
	domains.SaveIP(6)
}

func (domains *Domains) updateIPv6One(ctx context.Context, conf *config.Config, ipv6Addr string, dnsProvider string, wg *sync.WaitGroup, chanErrors chan error) error {
	dnsDomains := domains.IPv4Domains[dnsProvider]
	var _dnsDomains []*dnsdomain.Domain
	for _, dnsDomain := range dnsDomains {
		if dnsDomain == nil {
			continue
		}
		oldIP, err := resolver.ResolveDNS(dnsDomain.String(), conf.DNSResolver, `IPV6`)
		if err != nil {
			log.Errorf("[%s] ResolveDNS(%s): %s", dnsProvider, dnsDomain.String(), err.Error())
			//errs = append(errs, err)
			dnsDomain.UpdateStatus = dnsdomain.UpdatedIdle
			_dnsDomains = append(_dnsDomains, dnsDomain)
			continue
		}
		if oldIP != ipv6Addr {
			dnsDomain.UpdateStatus = dnsdomain.UpdatedIdle
			_dnsDomains = append(_dnsDomains, dnsDomain)
			continue
		}
		dnsDomain.UpdateStatus = dnsdomain.UpdatedNothing
		log.Infof("[%s] IP is the same as cached one (%s). Skip update (%s)", dnsProvider, ipv6Addr, dnsDomain.String())
	}
	if len(_dnsDomains) == 0 {
		return echo.ErrContinue
	}
	updater := ddnsmanager.Open(dnsProvider)
	if updater == nil {
		return echo.ErrContinue
	}
	dnsService := conf.FindService(dnsProvider)
	err := updater.Init(dnsService.Settings, _dnsDomains)
	if err != nil {
		chanErrors <- err
		return echo.ErrContinue
	}
	log.Infof("[%s] %s - Start to update record IP...", dnsProvider, ipv6Addr)
	err = updater.Update(ctx, `AAAA`, ipv6Addr)
	if err != nil {
		log.Errorf("[%s] %s - Failed to update IP: %v (Wait to try again later)", dnsProvider, ipv6Addr, err)
		wg.Add(1)
		go func() {
			defer wg.Done()
			chanErrors <- ddnsretry.Retry(ctx, func(retryCtx context.Context) error {
				err := updater.Update(retryCtx, `AAAA`, ipv6Addr)
				if err != nil {
					err = fmt.Errorf("[%s] %s - Failed to update IP: %v", dnsProvider, ipv6Addr, err)
				}
				return err
			})
		}()
		return echo.ErrContinue
	}
	return err
}

func (domains *Domains) makeUpdater(ipVer int, dnsProviders ...string) Updater {
	return func(ctx context.Context, conf *config.Config, ipAddr string, force bool) (changed bool, errs []error) {
		if len(ipAddr) == 0 {
			log.Warnf(`[DDNS] 没有查到ipv%d地址`, ipVer)
			return
		}
		if !force {
			if ipVer == 4 {
				if domains.IPv4Addr == ipAddr {
					return
				}
				log.Debugf(`[DDNS] 查询到ipv4变更: %s => %s`, domains.IPv4Addr, ipAddr)
			} else {
				if domains.IPv6Addr == ipAddr {
					return
				}
				log.Debugf(`[DDNS] 查询到ipv6变更: %s => %s`, domains.IPv6Addr, ipAddr)
			}
		}
		wg := &sync.WaitGroup{}
		chanErrors := make(chan error, len(dnsProviders))
		if ipVer == 4 {
			for _, dnsProvider := range dnsProviders {
				err := domains.updateIPv4One(ctx, conf, ipAddr, dnsProvider, wg, chanErrors)
				if err != nil {
					if err == echo.ErrContinue {
						continue
					}
					errs = append(errs, err)
					continue
				}
				changed = true
			}
		} else {
			for _, dnsProvider := range dnsProviders {
				err := domains.updateIPv6One(ctx, conf, ipAddr, dnsProvider, wg, chanErrors)
				if err != nil {
					if err == echo.ErrContinue {
						continue
					}
					errs = append(errs, err)
					continue
				}
				changed = true
			}
		}
		wg.Wait()
		close(chanErrors)
		for chanErr := range chanErrors {
			if chanErr == nil {
				changed = true
				continue
			}
			log.Error(chanErr.Error())
			errs = append(errs, chanErr)
		}
		if len(errs) == 0 {
			changed = true
		}
		return
	}
}

type Updater func(ctx context.Context, conf *config.Config, ipv6Addr string, force bool) (ipv6Changed bool, errs []error)

func (domains *Domains) Update(ctx context.Context, conf *config.Config, force bool, dnsProviders ...string) error {

	var (
		errs        []error
		ipv4Changed bool
		ipv6Changed bool
	)

	// IPv4
	if conf.IPv4.Enabled {
		ipv4Addr, err := utils.GetIPv4Addr(conf.IPv4)
		if err != nil {
			log.Error(err)
		} else {
			var updater Updater
			if len(dnsProviders) == 0 {
				for dnsProvider := range domains.IPv4Domains {
					dnsProviders = append(dnsProviders, dnsProvider)
				}
			}
			updater = domains.makeUpdater(4, dnsProviders...)
			ipv4Changed, errs = updater(ctx, conf, ipv4Addr, force)
			if ipv4Changed {
				domains.SetIPv4Addr(ipv4Addr)
			}
		}
	}
	// IPv6
	if conf.IPv6.Enabled {
		ipv6Addr, err := utils.GetIPv6Addr(conf.IPv6)
		if err != nil {
			log.Error(err)
		} else {
			var updater Updater
			if len(dnsProviders) == 0 {
				for dnsProvider := range domains.IPv6Domains {
					dnsProviders = append(dnsProviders, dnsProvider)
				}
			}
			updater = domains.makeUpdater(6, dnsProviders...)
			var _errs []error
			ipv6Changed, _errs = updater(ctx, conf, ipv6Addr, force)
			if ipv6Changed {
				domains.SetIPv6Addr(ipv6Addr)
			}
			if len(_errs) > 0 {
				if len(errs) > 0 {
					errs = append(errs, _errs...)
				} else {
					errs = _errs
				}
			}
		}
	}
	if !conf.IPv4.Enabled && !conf.IPv6.Enabled {
		return nil
	}
	var err error
	if len(errs) > 0 {
		errMessages := make([]string, len(errs))
		for index, err := range errs {
			errMessages[index] = err.Error()
		}
		err = errors.New(strings.Join(errMessages, "\n"))
	}
	var t *dnsdomain.TagValues
	tagValues := func() *dnsdomain.TagValues {
		if t != nil {
			return t
		}
		t = domains.TagValues(ipv4Changed, ipv6Changed)
		if err != nil {
			t.Error = err.Error()
		}
		return t
	}
	if conf.HasWebhook() {
		if err := conf.ExecWebhooks(tagValues()); err != nil {
			log.Errorf("[DDNS] webhook - %v", err)
		}
	}
	switch conf.NotifyMode {
	case config.NotifyDisabled:
		return err
	case config.NotifyIfError:
		if err == nil {
			return err
		}
	case config.NotifyAll:
	}
	if err := sender.Send(*tagValues(), conf.NotifyTemplate); err != nil {
		log.Errorf("[DDNS] sender.Send - %v", err)
	}
	return err
}
