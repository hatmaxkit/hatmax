# Generator Testing Plan - Road to 100% Coverage

**Current Status**: 1.6% coverage → Target: 100% coverage
**Scope**: Test the generator itself (meta-testing)

## 📊 Current Coverage Analysis

### ✅ **Tested Functions (4/48):**
- `InferRepoName` - 100.0%
- `InferMethodName` - 58.3% (needs edge cases)
- `InferRepoCall` - 100.0% 
- `capitalizeFirst` - 100.0%

### ❌ **Untested Functions (44/48 = 0.0% coverage):**

#### **Critical Core Functions** (High Priority):
- `NewModelGenerator` - Constructor, fundamental
- `GenerateAction` - Main entry point 
- `GenerateModels` - Core generation logic
- `GenerateRepoInterfaces` - Repository interfaces
- `GenerateSQLiteRepoImplementations` - SQLite repos
- `GenerateMongoRepoImplementations` - MongoDB repos  
- `GenerateAggregateModels` - Aggregate patterns

#### **Template & Data Functions** (High Priority):
- `buildSQLiteAggregateTemplateData` - Template data builder
- `buildMongoAggregateTemplateData` - Template data builder
- `buildRootFieldsData` - Root field processing
- `buildChildFieldsData` - Child field processing
- `generateFile` - File generation helper

#### **Utility Functions** (Medium Priority):
- `toSnakeCase` - String conversion
- `SanitizeName` - Name sanitization
- `mapGoType` - Type mapping
- `pluralize` - String pluralization  
- `sanitizeIdentifier` - Identifier cleaning
- `contains` - Array helper
- `isPartOfAggregate` - Logic helper
- `GetSQLiteDriver` - Config helper

#### **Service Generation** (Medium Priority):
- `GenerateServiceInterfaces` - Service interfaces
- `GenerateHandlers` - HTTP handlers
- `GenerateValidators` - Input validation
- `GenerateMain` - Main file generation
- `GenerateGoMod` - Go module files
- `GenerateMakefile` - Build system

#### **Infrastructure Functions** (Lower Priority):
- `GenerateConfigAndXParams` - Configuration
- `PostGenerationCleanup` - Cleanup tasks
- `Scaffold` - Project scaffolding
- `NewApp` - App initialization
- `UnmarshalYAML` - YAML processing
- `InferHandlerName` - Handler naming

#### **Deployment Functions** (Lower Priority):
- `NewDeploymentGenerator` - Deployment setup
- `GenerateNomadDeployments` - Nomad jobs
- `buildJobData` - Job configuration
- `loadTemplates` - Template loading
- `generateConfigTemplate` - Config templates
- `renderJobFile` - File rendering
- And more deployment-related functions...

## 🎯 **Testing Strategy**

### **Phase 1: Foundation (Target: 25% coverage)**
Focus on critical constructors and utility functions that other tests depend on:
- `NewModelGenerator` 
- `generateFile`
- `toSnakeCase`
- `mapGoType`
- `SanitizeName` 
- `contains`
- `isPartOfAggregate`

### **Phase 2: Core Generation (Target: 50% coverage)**
Test main generation functions with mock filesystems:
- `GenerateModels`
- `GenerateRepoInterfaces` 
- `GenerateAggregateModels`
- `buildSQLiteAggregateTemplateData`
- `buildMongoAggregateTemplateData`

### **Phase 3: Complete Pipeline (Target: 75% coverage)**
Test full generation pipeline:
- `GenerateAction` (main entry point)
- `GenerateSQLiteRepoImplementations`
- `GenerateMongoRepoImplementations`
- `GenerateHandlers`
- `GenerateValidators`

### **Phase 4: Edge Cases & Polish (Target: 100% coverage)**
Complete coverage with edge cases, error conditions, and remaining functions:
- All deployment functions
- Error handling paths
- Configuration edge cases
- YAML unmarshaling scenarios

## 🧪 **Testing Approach**

### **Tools & Techniques:**
- **Table-driven tests** for data transformation functions
- **Mock filesystem** (afero) for file generation tests
- **Golden files** for template output verification
- **Integration tests** for full generation pipeline
- **Error injection** for error path coverage

### **Test Structure:**
```
internal/hatmax/
├── config_test.go         ✅ (existing, fixed)
├── generator_test.go      📝 (create - core functions)
├── generate_test.go       📝 (create - generation pipeline) 
├── utils_test.go          📝 (create - utility functions)
├── deployment_test.go     📝 (create - deployment functions)
└── testdata/              📝 (create - golden files & fixtures)
    ├── configs/           📂 (test YAML configs)
    ├── expected/          📂 (expected generated output)
    └── templates/         📂 (template test data)
```

## 🚀 **Implementation Notes**

### **Challenges:**
- File system operations need mocking
- Template execution needs verification
- Complex data structures need deep comparison
- Error paths need systematic coverage

### **Success Metrics:**
- 100% function coverage
- 100% line coverage  
- 100% branch coverage
- All error paths tested
- All template outputs verified

---

**Status**: 📋 PLANNED - Ready for implementation when prioritized
**Estimated Effort**: Large (3-5 days for complete 100% coverage)
**Impact**: High (Ensures generator reliability and prevents regressions)