# SBOM Processor
This repository contains utility functions to process large amounts of SBOM related information and prepare them for further data analysis.

## Usage

### Transform Syft to CycloneDx
This command iterates through all json files in the given in directory and tries to parse them to a syft result struct. These structs are then transformed to cyclonedx SBOMs and stored in a file or a mongodb database depending on the chosen mode.
#### File to file transformation
```
go run cmd/transform/TransformSyft.go --mode file --in /path/to/your/sboms --out /path/to/store/sboms
```

#### File to database transformation
Database connection parameters are read from environment variables. How you set those is up to you. In the following example we temporarily set them in the command executing the go program.

```
MONGO_URI=mongodb://localhost:27017/dbname MONGO_USERNAME=USERNAME MONGO_PWD=PASSWORD go run cmd/transform/TransformSyft.go --mode db --in /path/to/your/sbom
```

### Export unique components
This command identifies all unique component names for a given programming language from the SBOMs and exports them to a file for further processing (e.g., metadata lookup for every component through [maven index search](https://github.com/fraunhofer-iem/maven-index-search)). 
```
MONGO_URI=mongodb://localhost:27017/dbname MONGO_USERNAME=USERNAME MONGO_PWD=PASSWORD go run cmd/export/ExportUniqueComponents.go --out /tmp/sboms
```