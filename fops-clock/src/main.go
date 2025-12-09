package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	envSchedule   = "FOPS_CLOCK_CRON_SCHEDULE"
	envCommand    = "FOPS_CLOCK_COMMAND"
	envShell      = "FOPS_CLOCK_CRON_SHELL"
	defaultShell  = "/bin/sh"
	logPrefix     = "[FOPS-CLOCK]"
	sleepAccuracy = time.Second
)

type config struct {
	Schedule string
	Command  string
	Shell    string
}

type overrides struct {
	Schedule string
	Command  string
	Shell    string
}

var shellDetector = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

var (
	flagSchedule = flag.String("schedule", "", "cron expression override")
	flagCommand  = flag.String("command", "", "command override")
	flagShell    = flag.String("shell", "", "shell path override")
)

func main() {
	flag.Parse()
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	cfg, err := loadConfig(overrides{
		Schedule: *flagSchedule,
		Command:  *flagCommand,
		Shell:    *flagShell,
	})
	if err != nil {
		logFatal(err)
	}

	schedule, err := parseSchedule(cfg.Schedule)
	if err != nil {
		logFatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go reapZombies(ctx)

	log.Printf("%s start schedule=%q command=%q shell=%q", logPrefix, cfg.Schedule, cfg.Command, cfg.Shell)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	for {
		delay := time.Until(schedule.Next(time.Now()))
		log.Printf("%s sleeping for %s", logPrefix, delay.Round(sleepAccuracy))
		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
			log.Printf("%s starting task", logPrefix)
			start := time.Now()
			if err := runJob(cfg.Command, cfg.Shell); err != nil {
				log.Printf("%s task failed after %s: %v", logPrefix, time.Since(start).Round(time.Millisecond), err)
			} else {
				log.Printf("%s task completed in %s", logPrefix, time.Since(start).Round(time.Millisecond))
			}
		case sig := <-sigChan:
			if !timer.Stop() {
				<-timer.C
			}
			log.Printf("%s received signal %s, shutting down", logPrefix, sig.String())
			cancel()
			return
		}
	}
}

func loadConfig(override overrides) (config, error) {
	schedule := strings.TrimSpace(override.Schedule)
	if schedule == "" {
		schedule = strings.TrimSpace(os.Getenv(envSchedule))
	}
	command := strings.TrimSpace(override.Command)
	if command == "" {
		command = strings.TrimSpace(os.Getenv(envCommand))
	}
	if schedule == "" {
		return config{}, fmt.Errorf("%s schedule is required via --schedule or %s", logPrefix, envSchedule)
	}
	if command == "" {
		return config{}, fmt.Errorf("%s command is required via --command or %s", logPrefix, envCommand)
	}

	shell := strings.TrimSpace(override.Shell)
	if shell == "" {
		shell = strings.TrimSpace(os.Getenv(envShell))
	}
	if shell == "" {
		shell = defaultShell
	}

	return config{
		Schedule: schedule,
		Command:  command,
		Shell:    shell,
	}, nil
}

func parseSchedule(spec string) (cron.Schedule, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(spec)
	if err != nil {
		return nil, fmt.Errorf("%s invalid cron expression: %w", logPrefix, err)
	}
	return schedule, nil
}

func runJob(command, shellPath string) error {
	cmd, err := buildCommand(command, shellPath)
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func buildCommand(command, shellPath string) (*exec.Cmd, error) {
	if command == "" {
		return nil, errors.New("command cannot be empty")
	}

	if shellPath != "" && shellDetector(shellPath) {
		return exec.Command(shellPath, "-c", command), nil
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, errors.New("invalid command")
	}

	return exec.Command(parts[0], parts[1:]...), nil
}

func reapZombies(ctx context.Context) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGCHLD)
	defer signal.Stop(sigCh)

	for {
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			drainZombies()
		}
	}
}

func drainZombies() {
	for {
		var status syscall.WaitStatus
		_, err := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if err != nil {
			if errors.Is(err, syscall.ECHILD) || errors.Is(err, syscall.EINVAL) {
				return
			}
			if errors.Is(err, syscall.EINTR) {
				continue
			}
			return
		}
	}
}

func logFatal(err error) {
	log.Printf("%s fatal: %v", logPrefix, err)
	os.Exit(1)
}
