package main

import (
	"context"
	"fmt"

	"github.com/creativeprojects/go-selfupdate"
)

func selfUpgrade() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "SHA256SUMS"},
	})
	if err != nil {
		return false, err
	}

	release, found, err := updater.DetectLatest(ctx, selfupdate.NewRepositorySlug("derekn", "cfddns"))
	if err != nil {
		return false, err
	}
	if !found {
		return false, fmt.Errorf("no releases found")
	}

	if release.LessOrEqual(version) {
		fmt.Println("Already on latest release")
		return false, nil
	}
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return false, err
	}
	if err := updater.UpdateTo(ctx, release, exe); err != nil {
		return false, err
	}
	fmt.Println(exe)
	fmt.Printf("Updated to version %s\n", release.Version())
	return true, nil
}
