# Prompt Package Specification (.pbt)

* Version: 0.1.0  
* Schema: ppkg.schema.json

A .pbt archive **MUST** contain:

1. ppkg.json (manifest, validated by schema)  
2. One or more prompt templates listed in `promptFiles`  
3. Optional `tests/` directory (Promptfoo YAML)  
4. Optional assets (images) referenced by README