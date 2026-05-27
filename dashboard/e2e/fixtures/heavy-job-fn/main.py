import json
import os

input_data = json.loads(os.environ.get("INPUT", "{}"))
result = {
    "kind": "heavy-job",
    "processed": True,
    "items": len(input_data.get("items", [])),
}
print(json.dumps(result))
