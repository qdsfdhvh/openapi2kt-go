package testutil

import "encoding/json"

// BooleanHTTPControl lowers only direct JSON request/success-response roots.
// It is test tooling; SSE envelope/data, error schemas and metadata stay intact.
func BooleanHTTPControl(input []byte) ([]byte, error) {
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(input, &doc); err != nil {
		return nil, err
	}
	var paths map[string]map[string]json.RawMessage
	if err := json.Unmarshal(doc["paths"], &paths); err != nil {
		return nil, err
	}
	for _, path := range paths {
		for method, raw := range path {
			var op map[string]json.RawMessage
			if err := json.Unmarshal(raw, &op); err != nil {
				return nil, err
			}
			if body := op["requestBody"]; body != nil {
				lowered, err := booleanHTTPContentControl(body)
				if err != nil {
					return nil, err
				}
				op["requestBody"] = lowered
			}
			var responses map[string]json.RawMessage
			if err := json.Unmarshal(op["responses"], &responses); err != nil {
				return nil, err
			}
			for status, response := range responses {
				if status[0] == '2' {
					lowered, err := booleanHTTPContentControl(response)
					if err != nil {
						return nil, err
					}
					responses[status] = lowered
				}
			}
			op["responses"], _ = json.Marshal(responses)
			path[method], _ = json.Marshal(op)
		}
	}
	doc["paths"], _ = json.Marshal(paths)
	var components map[string]json.RawMessage
	if err := json.Unmarshal(doc["components"], &components); err != nil {
		return nil, err
	}
	for _, key := range []string{"requestBodies", "responses"} {
		var slots map[string]json.RawMessage
		if err := json.Unmarshal(components[key], &slots); err != nil {
			return nil, err
		}
		for name, raw := range slots {
			lowered, err := booleanHTTPContentControl(raw)
			if err != nil {
				return nil, err
			}
			slots[name] = lowered
		}
		components[key], _ = json.Marshal(slots)
	}
	doc["components"], _ = json.Marshal(components)
	return json.Marshal(doc)
}
func booleanHTTPContentControl(raw json.RawMessage) (json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil {
		return nil, err
	}
	if content := object["content"]; content != nil {
		var media map[string]map[string]json.RawMessage
		if err := json.Unmarshal(content, &media); err != nil {
			return nil, err
		}
		for name, entry := range media {
			if name != "text/event-stream" && string(entry["schema"]) == "true" {
				entry["schema"] = json.RawMessage(`{"type":"object"}`)
			}
		}
		object["content"], _ = json.Marshal(media)
	}
	return json.Marshal(object)
}
