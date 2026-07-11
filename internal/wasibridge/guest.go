package wasibridge

import (
	"fmt"
	"io"
	"os"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// GuestLookupCredential is what the WASM guest calls in place of talking to
// Postgres directly. It writes a store_call frame to stdout, blocks on the
// host's response frame from stdin, and reconstructs the store result. This is
// the seam a generated bridge would fill for every store method; the spike
// implements it by hand for one.
//
// It is written against os.Stdin/os.Stdout (not passed in) because that is the
// only channel a wasip1 guest shares with its host. A transport error is
// surfaced as a rejected result rather than a panic so the guest can still
// report a well-formed frame.
func GuestLookupCredential(email string) auth.CredentialLookupResult {
	return guestLookupCredential(os.Stdout, os.Stdin, email)
}

func guestLookupCredential(out io.Writer, in io.Reader, email string) auth.CredentialLookupResult {
	request := guestFrame{Kind: "store_call", Call: &lookupRequest{
		Store:  "auth",
		Method: "FindCredentialByEmail",
		Email:  email,
	}}
	if err := writeGuestFrame(out, request); err != nil {
		return rejected(core.ErrorCodeInvalidState, "guest: send store call: "+err.Error())
	}

	response, err := readHostFrame(in)
	if err != nil {
		return rejected(core.ErrorCodeInvalidState, "guest: read store response: "+err.Error())
	}
	if response.Response == nil {
		return rejected(core.ErrorCodeInvalidState, "guest: empty store response")
	}
	return credentialFromWire(*response.Response)
}

// GuestReportResult writes the guest's final answer for the unit of work as a
// result frame. The spike host reads it back and compares it to a direct store
// call to prove the value survived the guest boundary unchanged.
func GuestReportResult(result auth.CredentialLookupResult) error {
	frame := guestFrame{Kind: "result"}
	wire := credentialToWire(result)
	frame.Result = &wire
	if err := writeGuestFrame(os.Stdout, frame); err != nil {
		return fmt.Errorf("guest: report result: %w", err)
	}
	return nil
}
