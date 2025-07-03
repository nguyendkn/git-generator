# Git Generator

[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

Git Generator is a professional CLI tool that uses AI to generate high-quality Git commit messages based on your staged changes. It analyzes your diff and creates conventional commit messages that follow best practices.

Test bullet points formatting in commit messages.

## Features

- 🤖 **AI-Powered**: Uses Google Gemini API for intelligent commit message generation
- 📝 **Conventional Commits**: Follows conventional commit standards by default
- 🔍 **Smart Diff Analysis**: Intelligently parses and chunks large diffs
- 🎨 **Multiple Styles**: Support for conventional, simple, and detailed commit message styles
- 🔧 **Configurable**: Extensive configuration options via YAML config file
- 🚀 **Fast & Reliable**: Built with Go for performance and reliability
- 📊 **Language Detection**: Automatically detects programming languages in changes
- 🔄 **Interactive Mode**: Preview and edit commit messages before applying
- 📋 **Dry Run**: Preview commit messages without applying them

## Installation

### Prerequisites
- Go 1.21 or later
- Git repository
- Google Gemini API key

### From Source

```bash
git clone https://github.com/nguyendkn/git-generator.git
cd git-generator
go build -o git-generator cmd/git-generator/main.go
```

### Using Go Install

```bash
go install github.com/nguyendkn/git-generator/cmd/git-generator@latest
```

## Quick Start

1. **Initialize configuration**:
   ```bash
   git-generator init
   ```

2. **Set your Gemini API key**:
   ```bash
   git-generator config set-api-key YOUR_API_KEY
   ```
   
   Or set the environment variable:
   ```bash
   export GEMINI_API_KEY=your_api_key_here
   ```

3. **Stage your changes**:
   ```bash
   git add .
   ```

4. **Generate and apply commit message**:
   ```bash
   git-generator generate
   ```

## Usage

### Basic Commands

```bash
# Generate commit message for staged changes
git-generator generate

# Preview without committing (dry-run)
git-generator generate --dry-run

# Generate multiple options
git-generator generate --multiple

# Use different style
git-generator generate --style simple

# Show repository status
git-generator status

# Show current configuration
git-generator config show
```

### Command Options

#### `generate` command

- `--style, -s`: Commit message style (`conventional`, `simple`, `detailed`)
- `--dry-run, -d`: Preview the commit message without applying it
- `--staged, -S`: Use staged changes (default: true)
- `--multiple, -m`: Generate multiple commit message options

#### `config` command

- `show`: Display current configuration
- `set-api-key [key]`: Set the Gemini API key

## Configuration

Git Generator uses a YAML configuration file located at `~/.git-generator/git-generator.yaml`.

### Example Configuration

```yaml
# Git Generator Configuration

gemini:
  # Get your API key from: https://makersuite.google.com/app/apikey
  api_key: "your_api_key_here"
  model: "gemini-1.5-flash"
  temperature: 0.3
  max_tokens: 1000

git:
  max_diff_size: 10000
  include_staged: true
  ignore_files:
    - "*.log"
    - "*.tmp"
    - "node_modules/*"
    - ".git/*"
    - "vendor/*"
    - "target/*"
    - "build/*"
    - "dist/*"

output:
  style: "conventional"  # conventional, simple, detailed
  max_lines: 100
  dry_run: false
```

### Configuration Options

#### Gemini Settings

- `api_key`: Your Google Gemini API key (required)
- `model`: Gemini model to use (default: "gemini-1.5-flash")
- `temperature`: AI creativity level 0.0-2.0 (default: 0.3)
- `max_tokens`: Maximum response length (default: 1000)

#### Git Settings

- `max_diff_size`: Maximum diff size to process (default: 10000)
- `include_staged`: Include staged changes (default: true)
- `ignore_files`: File patterns to ignore

#### Output Settings

- `style`: Default commit message style (default: "conventional")
- `max_lines`: Maximum lines in output (default: 100)
- `dry_run`: Default to dry-run mode (default: false)

## Commit Message Styles

### Conventional (Default)

Follows the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
feat(auth): add JWT authentication

Implement JWT-based authentication system with refresh token support.
Includes middleware for route protection and token validation.

Closes #123
```

### Simple

Concise, single-line commit messages:

```
Add user authentication system
```

### Detailed

Comprehensive commit messages with full context:

```
Implement comprehensive user authentication system

This commit introduces a complete authentication system including:
- JWT token generation and validation
- Refresh token mechanism
- Password hashing with bcrypt
- Session management
- Route protection middleware

The system supports both login and registration workflows
and includes proper error handling and validation.

Files changed:
- auth/jwt.go: JWT token handling
- auth/middleware.go: Authentication middleware
- handlers/auth.go: Authentication endpoints
- models/user.go: User model updates

Closes #123
Fixes #124
```

## Examples

### Basic Usage

```bash
# Stage your changes
git add src/auth.go src/middleware.go

# Generate and apply commit message
git-generator generate
# Output: feat(auth): implement JWT authentication system
```

### Preview Mode

```bash
git-generator generate --dry-run
```

Output:
```
Preview (dry-run mode):
Commit Message:
feat(auth): implement JWT authentication system

Implement JWT-based authentication with middleware support.
Includes token validation and refresh mechanisms.

Changes Summary:
2 added files with 150 additions and 0 deletions. Languages: Go (2)

Languages:
  - Go: 2 files

File Changes:
  1. added: src/auth.go (75+, 0-)
  2. added: src/middleware.go (75+, 0-)
```

### Multiple Options

```bash
git-generator generate --multiple
```

Output:
```
Generated commit message options:

1. feat(auth): implement JWT authentication system

2. add: JWT authentication and middleware

3. feat: implement comprehensive authentication system

   Add JWT-based authentication with token validation,
   refresh mechanisms, and route protection middleware.
```

## API Key Setup

### Get Your API Key

1. Visit [Google AI Studio](https://makersuite.google.com/app/apikey)
2. Create a new API key
3. Copy the key

### Set the API Key

**Option 1: Using the CLI**
```bash
git-generator config set-api-key YOUR_API_KEY
```

**Option 2: Environment Variable**
```bash
export GEMINI_API_KEY=your_api_key_here
```

**Option 3: Configuration File**
Edit `~/.git-generator/git-generator.yaml` and add:
```yaml
gemini:
  api_key: "your_api_key_here"
```

## Development

### Building from Source

```bash
git clone https://github.com/nguyendkn/git-generator.git
cd git-generator
go mod download
go build -o git-generator cmd/git-generator/main.go
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
git-generator/
├── cmd/git-generator/     # Main CLI application
├── internal/              # Internal packages
│   ├── ai/               # Gemini API integration
│   ├── config/           # Configuration management
│   ├── diff/             # Diff processing and chunking
│   ├── generator/        # Core generation logic
│   ├── git/              # Git operations
│   └── logger/           # Logging utilities
├── pkg/types/            # Public types and interfaces
└── README.md
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Conventional Commits](https://www.conventionalcommits.org/) for the commit message standard
- [Google Gemini](https://ai.google.dev/) for the AI capabilities
- [Cobra](https://github.com/spf13/cobra) for the CLI framework
- [Viper](https://github.com/spf13/viper) for configuration management
# Test formatting and validation
