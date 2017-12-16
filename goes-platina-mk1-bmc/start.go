// Copyright © 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by the GPL-2 license described in the
// LICENSE file.

package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/platinasystems/go/goes/cmd"
	"github.com/platinasystems/go/goes/cmd/platina/mk1/bmc/upgrade"
	"github.com/platinasystems/go/goes/cmd/start"
	"github.com/platinasystems/go/internal/gpio"
	"github.com/platinasystems/go/internal/log"
	"github.com/platinasystems/go/internal/redis"
	"github.com/platinasystems/go/internal/redis/publisher"
)

func startInit() {
	start.ConfGpioHook = func() error {
		var deviceVer byte

		cmd.Init("gpio")
		pin, found := gpio.Pins["QSPI_MUX_SEL"]
		if found {
			r, _ := pin.Value()
			if r {
				log.Print("Booted from QSPI1")
			} else {
				log.Print("Booted from QSPI0")
			}

		}

		for name, pin := range gpio.Pins {
			err := pin.SetDirection()
			if err != nil {
				fmt.Printf("%s: %v\n", name, err)
			}
		}
		pin, found = gpio.Pins["FRU_I2C_MUX_RST_L"]
		if found {
			pin.SetValue(false)
			time.Sleep(1 * time.Microsecond)
			pin.SetValue(true)
		}

		pin, found = gpio.Pins["MAIN_I2C_MUX_RST_L"]
		if found {
			pin.SetValue(false)
			time.Sleep(1 * time.Microsecond)
			pin.SetValue(true)
		}
		redis.Hwait(redis.DefaultHash, "redis.ready", "true",
			10*time.Second)

		ss, _ := redis.Hget(redis.DefaultHash, "eeprom.DeviceVersion")
		_, _ = fmt.Sscan(ss, &deviceVer)
		if deviceVer == 0x0 || deviceVer == 0xff {
			pin, found = gpio.Pins["FP_BTN_UARTSEL_EN_L"]
			if found {
				pin.SetValue(false)
			}
		}
		if err := pubEth0(); err != nil {
			return err
		}
		upgrade.UpdateEnv(false)
		upgrade.UpdateEnv(true)
		return nil
	}
}

func pubEth0() (err error) {
	var last, kmsg log.Kmsg
	f, err := os.Open("/dev/kmsg")
	if err != nil {
		return err
	}
	defer f.Close()
	if err = syscall.SetNonblock(int(f.Fd()), true); err != nil {
		return err
	}
	buf := make([]byte, 4096)
	defer func() { buf = buf[:0] }()
	for {
		n, err := f.Read(buf)
		if err != nil {
			break
		}
		kmsg.Parse(buf[:n])
		if last.Seq == 0 || last.Seq < kmsg.Seq {
			if strings.Contains(kmsg.Msg, "init.redisd") {
				if strings.Contains(kmsg.Msg, "eth0") {
					err := pubAddr(kmsg.Msg)
					if err != nil {
						return err
					}
				}
				last.Seq = kmsg.Seq
			}
		}
	}
	return nil
}

func pubAddr(s string) (err error) {
	ip := strings.SplitAfter(s, "[")
	i := ip[2]
	ip = strings.Split(i, "%")
	if strings.Contains(s, "::") {
		err = pubKey("eth0.ipv6", ip[0])
		if err != nil {
			return err
		}
	} else {
		err = pubKey("eth0.ipv4", ip[0])
		if err != nil {
			return err
		}
	}
	return nil
}

func pubKey(k string, v interface{}) (err error) {
	var pub *publisher.Publisher
	if pub, err = publisher.New(); err != nil {
		return err
	}
	pub.Print(k, ": ", v)
	return nil
}
