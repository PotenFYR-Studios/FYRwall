package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

// decodeStrict enforces strict JSON decoding: unknown fields rejected,
// single JSON value, size already capped by bodyLimit middleware
// (spec sections 35, 77).
func decodeStrict(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errors.New("request body too large")
		}
		return errors.New("malformed JSON body: " + err.Error())
	}
	// Reject trailing content.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing data after JSON body")
	}
	return nil
}

// ValidateTx exposes transaction validation through the manager for the
// validate endpoint.
func (s *Server) ValidateTx(ctx context.Context, tx firewall.Transaction) firewall.ValidationResult {
	return s.firewall.BackendValidate(ctx, tx)
}
