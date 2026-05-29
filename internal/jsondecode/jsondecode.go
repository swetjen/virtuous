package jsondecode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// Options configures JSON request decoding behavior.
type Options struct {
	DisallowUnknownFields  bool
	DisallowDuplicateKeys  bool
	DisallowTrailingTokens bool
}

// StrictOptions returns the strict request-decoding profile used by Virtuous.
func StrictOptions() Options {
	return Options{
		DisallowUnknownFields:  true,
		DisallowDuplicateKeys:  true,
		DisallowTrailingTokens: true,
	}
}

// DuplicateKeyError reports a repeated JSON object key.
type DuplicateKeyError struct {
	Key string
}

func (e DuplicateKeyError) Error() string {
	return "json: duplicate object key " + strconv.Quote(e.Key)
}

// Decode decodes one JSON value from r into v using opts.
func Decode(r io.Reader, v any, opts Options) error {
	if opts.DisallowDuplicateKeys {
		data, err := io.ReadAll(r)
		if err != nil {
			return err
		}
		if err := rejectDuplicateKeys(data, opts.DisallowTrailingTokens); err != nil {
			return err
		}
		r = bytes.NewReader(data)
	}

	dec := json.NewDecoder(r)
	if opts.DisallowUnknownFields {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(v); err != nil {
		return err
	}
	if opts.DisallowTrailingTokens {
		return rejectTrailingTokens(dec)
	}
	return nil
}

func rejectDuplicateKeys(data []byte, disallowTrailing bool) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := scanValue(dec); err != nil {
		return err
	}
	if disallowTrailing {
		return rejectTrailingTokens(dec)
	}
	return nil
}

func scanValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}

	switch delim {
	case '{':
		return scanObject(dec)
	case '[':
		return scanArray(dec)
	default:
		return fmt.Errorf("json: unexpected delimiter %q", delim)
	}
}

func scanObject(dec *json.Decoder) error {
	seen := map[string]struct{}{}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := tok.(string)
		if !ok {
			return fmt.Errorf("json: object key is not a string")
		}
		if _, exists := seen[key]; exists {
			return DuplicateKeyError{Key: key}
		}
		seen[key] = struct{}{}
		if err := scanValue(dec); err != nil {
			return err
		}
	}
	return expectDelimiter(dec, '}')
}

func scanArray(dec *json.Decoder) error {
	for dec.More() {
		if err := scanValue(dec); err != nil {
			return err
		}
	}
	return expectDelimiter(dec, ']')
}

func expectDelimiter(dec *json.Decoder, want json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := tok.(json.Delim)
	if !ok || delim != want {
		return fmt.Errorf("json: expected delimiter %q", want)
	}
	return nil
}

func rejectTrailingTokens(dec *json.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}
	return errors.New("json: trailing data after top-level value")
}
