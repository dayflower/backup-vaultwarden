package main

import (
	"fmt"
	"os"
)

func handleDeferError(err error, f func() error) error {
	if e := f(); e != nil {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w; %v", err, e)
	}
	return err
}

func tempFileName(pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}

	name := f.Name()

	if err = f.Close(); err != nil {
		return "", err
	}

	if err = os.Remove(name); err != nil {
		return "", err
	}

	return name, nil
}
