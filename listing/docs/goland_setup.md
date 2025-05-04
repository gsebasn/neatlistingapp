# GoLand Setup Guide

This guide will help you set up GoLand to work with this project, including linting and debugging configuration.

## Initial Setup

1. Open the project in GoLand
2. Go to `File > Settings > Go > GOPATH` and ensure your GOPATH is correctly set
3. Go to `File > Settings > Go > Go Modules (vgo)` and ensure Go modules are enabled

## Linting Configuration

1. Go to `File > Settings > Editor > Inspections`
2. Click on the gear icon next to "Profile" and select "Import Profile"
3. Navigate to `.idea/inspectionProfiles/Project_Default.xml` and import it
4. Click "Apply" and "OK"

## Run Configuration

1. Go to `Run > Edit Configurations`
2. Click the "+" button and select "Go Build"
3. Configure the following:
   - Name: `ListingApp`
   - Run kind: `File`
   - File: `main.go`
   - Working directory: `$PROJECT_DIR$`
   - Environment: `APP_ENV=dev`

## Debugging Setup

1. In the run configuration, ensure "Build before run" is checked
2. Set breakpoints by clicking in the gutter (left margin) of the code editor
3. Use the debug toolbar to:
   - Start debugging (bug icon)
   - Step over (F8)
   - Step into (F7)
   - Step out (Shift+F8)
   - Resume program (F9)

## Running the Application

### Normal Run
1. Click the green "Run" button or press Shift+F10
2. The application will start with the development environment

### Debug Run
1. Click the green "Debug" button or press Shift+F9
2. The application will start in debug mode
3. Use the debug toolbar to control execution
4. View variables in the "Variables" window
5. Use the "Watches" window to monitor specific expressions

## Linting in GoLand

The project uses several linters that are integrated into GoLand:

- gofmt: Code formatting
- govet: Common programming mistakes
- staticcheck: Advanced static analysis
- gosimple: Code simplification suggestions
- ineffassign: Ineffective assignments
- unused: Unused code detection
- errcheck: Error handling checks
- misspell: Spelling mistakes
- gosec: Security checks

You can run the linters:
1. On save (configured in the inspection profile)
2. Manually via `Code > Inspect Code`
3. Using the shell script: `./scripts/lint.sh`

## Environment Variables

The application uses environment variables from `env.dev` in development. GoLand will use these when running the application through the IDE.

## Troubleshooting

If you encounter issues:

1. Ensure all Go tools are installed:
   ```bash
   go install golang.org/x/tools/cmd/...@latest
   go install honnef.co/go/tools/cmd/staticcheck@latest
   ```

2. If linting doesn't work:
   - Go to `File > Settings > Tools > File Watchers`
   - Ensure the Go tools watchers are enabled

3. If debugging doesn't work:
   - Ensure you're using a compatible version of Go
   - Check that the debug configuration is properly set up
   - Try invalidating caches and restarting GoLand 