package forest

import "errors"

var ErrApiTokenEnvVarNotSet = errors.New("FOREST_API_TOKEN environment variable is not set")
