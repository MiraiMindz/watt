package http11

// ParseMethodID converts an HTTP method byte slice to a numeric ID.
// Returns MethodUnknown for unrecognized methods.
// This function performs zero allocations and uses O(1) byte-level comparisons.
//
// Allocation behavior: 0 allocs/op
func ParseMethodID(method []byte) uint8 {
	// Fast path: check length first to reduce comparisons
	switch len(method) {
	case 3: // GET, PUT
		if method[0] == 'G' && method[1] == 'E' && method[2] == 'T' {
			return MethodGET
		}
		if method[0] == 'P' && method[1] == 'U' && method[2] == 'T' {
			return MethodPUT
		}

	case 4: // POST, HEAD
		if method[0] == 'P' && method[1] == 'O' && method[2] == 'S' && method[3] == 'T' {
			return MethodPOST
		}
		if method[0] == 'H' && method[1] == 'E' && method[2] == 'A' && method[3] == 'D' {
			return MethodHEAD
		}

	case 5: // PATCH, TRACE
		if method[0] == 'P' && method[1] == 'A' && method[2] == 'T' && method[3] == 'C' && method[4] == 'H' {
			return MethodPATCH
		}
		if method[0] == 'T' && method[1] == 'R' && method[2] == 'A' && method[3] == 'C' && method[4] == 'E' {
			return MethodTRACE
		}

	case 6: // DELETE
		if method[0] == 'D' && method[1] == 'E' && method[2] == 'L' &&
			method[3] == 'E' && method[4] == 'T' && method[5] == 'E' {
			return MethodDELETE
		}

	case 7: // OPTIONS, CONNECT
		if method[0] == 'O' && method[1] == 'P' && method[2] == 'T' &&
			method[3] == 'I' && method[4] == 'O' && method[5] == 'N' && method[6] == 'S' {
			return MethodOPTIONS
		}
		if method[0] == 'C' && method[1] == 'O' && method[2] == 'N' &&
			method[3] == 'N' && method[4] == 'E' && method[5] == 'C' && method[6] == 'T' {
			return MethodCONNECT
		}
	}

	return MethodUnknown
}

// MethodString returns the string representation of a method ID.
// Uses pre-compiled constants for zero allocations.
//
// Allocation behavior: 0 allocs/op
func MethodString(id uint8) string {
	switch id {
	case MethodGET:
		return methodGETString
	case MethodPOST:
		return methodPOSTString
	case MethodPUT:
		return methodPUTString
	case MethodDELETE:
		return methodDELETEString
	case MethodPATCH:
		return methodPATCHString
	case MethodHEAD:
		return methodHEADString
	case MethodOPTIONS:
		return methodOPTIONSString
	case MethodCONNECT:
		return methodCONNECTString
	case MethodTRACE:
		return methodTRACEString
	default:
		return ""
	}
}

// MethodBytes returns the byte slice representation of a method ID.
// Uses pre-compiled constants for zero allocations.
//
// Allocation behavior: 0 allocs/op
func MethodBytes(id uint8) []byte {
	switch id {
	case MethodGET:
		return methodGETBytes
	case MethodPOST:
		return methodPOSTBytes
	case MethodPUT:
		return methodPUTBytes
	case MethodDELETE:
		return methodDELETEBytes
	case MethodPATCH:
		return methodPATCHBytes
	case MethodHEAD:
		return methodHEADBytes
	case MethodOPTIONS:
		return methodOPTIONSBytes
	case MethodCONNECT:
		return methodCONNECTBytes
	case MethodTRACE:
		return methodTRACEBytes
	default:
		return nil
	}
}

// IsValidMethodID checks if a method ID is valid (not MethodUnknown).
// Allocation behavior: 0 allocs/op
func IsValidMethodID(id uint8) bool {
	return id >= MethodGET && id <= MethodTRACE
}
