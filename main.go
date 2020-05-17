package main

import (
	"os"
	"runtime"
	"runtime/debug"
	"runtime/trace"

	"github.com/diamondburned/gtkcord3/gtkcord"
	"github.com/diamondburned/gtkcord3/gtkcord/components/login"
	"github.com/diamondburned/gtkcord3/gtkcord/ningen"
	"github.com/diamondburned/gtkcord3/gtkcord/semaphore"
	"github.com/diamondburned/gtkcord3/internal/keyring"
	"github.com/diamondburned/gtkcord3/internal/log"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"

	// Profiler
	"net/http"
	_ "net/http/pprof"
)

var profile bool

func init() {
	// flag.BoolVar(&profile, "prof", false, "Enable the profiler")

	// AGGRESSIVE GC
	debug.SetGCPercent(100)

	// Set the right envs:
	LoadEnvs()

	debug.SetTraceback("all")

	f, err := os.OpenFile("/tmp/gtkcord-trace", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0750)
	if err != nil {
		log.Fatalln("Failed to open trace file:", err)
	}

	if err := trace.Start(f); err != nil {
		log.Fatalln("Failed to start trace:", err)
	}
}

func LoadToken() string {
	var token = os.Getenv("TOKEN")
	if token != "" {
		return token
	}

	// flag.StringVar(&token, "t", "", "Token")
	// flag.Parse()

	return token
}

func LoadKeyring() string {
	// If $TOKEN or -t is provided, override it in the keyring and use it:
	if token := LoadToken(); token != "" {
		return token
	}

	// Does the keyring have the token? Maybe.
	return keyring.Get()
}

func Login(finish func(s *ningen.State)) error {
	var lastErr error
	var token = LoadKeyring()

	if token != "" {
		_, err := ningen.Connect(token, finish)
		if err == nil {
			return nil
		}

		log.Errorln("Failed to re-use token:", err)
		lastErr = err
	}

	// No, so we need to display the login window:
	semaphore.IdleMust(func() {
		l := login.NewLogin(finish)
		l.LastError = lastErr
		l.LastToken = token

		l.Run()
	})

	return nil
}

func Finish(a *gtkcord.Application) func(s *ningen.State) {
	return func(s *ningen.State) {
		if err := a.Ready(s); err != nil {
			log.Fatalln("Failed to get gtkcord ready:", err)
		}
	}
}

func main() {
	a, err := gtk.ApplicationNew("com.github.diamondburned.gtkcord3", glib.APPLICATION_FLAGS_NONE)
	if err != nil {
		log.Fatalln("Failed to create a new *gtk.Application:", err)
	}

	g := gtkcord.New(a)

	a.Connect("activate", func() {
		g.Activate()

		go func() {
			// Try and log in:
			if err := Login(Finish(g)); err != nil {
				log.Fatalln("Failed to login:", err)
			}
		}()
	})

	a.Connect("shutdown", func() {
		g.Close()
	})

	if profile {
		// Profiler
		runtime.SetMutexProfileFraction(5)   // ???
		runtime.SetBlockProfileRate(5000000) // 5ms
		go http.ListenAndServe("localhost:6969", nil)
	}

	if sig := a.Run(os.Args); sig > 0 {
		os.Exit(sig)
	}
}
