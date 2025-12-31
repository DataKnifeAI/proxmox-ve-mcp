package proxmox

import (
	"encoding/json"
)

// boolToInt converts boolean to int (0 or 1)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// unmarshalData converts interface{} to specific type
func (c *Client) unmarshalData(data interface{}, result interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, result)
}

// marshalJSON is a helper to convert interface{} to JSON bytes
func marshalJSON(data interface{}) []byte {
	jsonData, _ := json.Marshal(data)
	return jsonData
}
