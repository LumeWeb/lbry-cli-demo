package shared

// Account service endpoint constants
const (
	// Authentication endpoints
	EndpointAuthRegister    = "/api/auth/register"
	EndpointAuthLogin       = "/api/auth/login"
	EndpointAuthOTPGenerate = "/api/auth/otp/generate"
	EndpointAuthOTPVerify   = "/api/auth/otp/verify"
	EndpointAuthOTPValidate = "/api/auth/otp/validate"

	// Account endpoints
	EndpointAccountKeys = "/api/account/keys"
)

// LBRY service endpoint constants
const (
	// Device endpoints
	LBRYEndpointDevices = "/api/devices"

	// Stream endpoints
	LBRYEndpointStreams      = "/api/streams"
	LBRYEndpointStreamUpload = "/api/streams/upload"
	LBRYEndpointStreamPin    = "/api/streams/pin"

	// Operations endpoints
	EndpointOperations = "/api/operations"
)
