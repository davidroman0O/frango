# Contributing to Frango

Thank you for your interest in contributing to Frango! This guide will help you understand how to contribute to the project effectively.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Code Style Guidelines](#code-style-guidelines)
- [Testing Guidelines](#testing-guidelines)
- [Documentation Guidelines](#documentation-guidelines)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting Guidelines](#issue-reporting-guidelines)
- [Community and Communication](#community-and-communication)

## Code of Conduct

Our community strives to be open, inclusive, and respectful. We expect all contributors to adhere to these principles:

- Be respectful and inclusive of differing viewpoints and experiences
- Focus on what is best for the community and the project
- Show empathy towards other community members
- Accept constructive criticism gracefully
- Avoid personal attacks or derogatory comments

## Getting Started

### Setting Up Your Development Environment

1. **Fork the Repository**

   Start by forking the [Frango repository](https://github.com/davidroman0O/go-php) on GitHub.

2. **Clone Your Fork**

   ```bash
   git clone https://github.com/YOUR-USERNAME/go-php.git
   cd go-php
   ```

3. **Set Up Remote**

   Add the original repository as an upstream remote:

   ```bash
   git remote add upstream https://github.com/davidroman0O/go-php.git
   ```

4. **Install Dependencies**

   Make sure you have Go installed (version 1.16+) and PHP with php-cgi:

   ```bash
   # Check Go version
   go version
   
   # Check PHP and php-cgi
   php --version
   php-cgi --version
   ```

5. **Run Tests**

   Ensure the test suite passes on your system:

   ```bash
   go test ./...
   ```

### Finding Issues to Work On

- Check for issues labeled as `good first issue` or `help wanted` in the [GitHub issue tracker](https://github.com/davidroman0O/go-php/issues)
- Look for TODOs in the codebase
- Suggest improvements by opening a new issue

## Development Workflow

### Branch Strategy

- `main` - stable release branch
- `develop` - ongoing development branch
- Feature branches - named like `feature/your-feature-name`
- Bugfix branches - named like `fix/issue-description`

### Working on a New Feature or Bug Fix

1. **Create a New Branch**

   Always create a new branch from the latest `develop`:

   ```bash
   git checkout develop
   git pull upstream develop
   git checkout -b feature/my-new-feature
   ```

2. **Commit Your Changes**

   Make your changes, then commit with a clear message:

   ```bash
   git add .
   git commit -m "feat: add new functionality for X"
   ```

   We follow [Conventional Commits](https://www.conventionalcommits.org/) for commit messages:

   - `feat:` - A new feature
   - `fix:` - A bug fix
   - `docs:` - Documentation changes
   - `style:` - Code style changes (formatting, etc.)
   - `refactor:` - Code changes that neither fix bugs nor add features
   - `perf:` - Performance improvements
   - `test:` - Adding or correcting tests
   - `chore:` - Changes to the build process or auxiliary tools

3. **Push Your Changes**

   ```bash
   git push origin feature/my-new-feature
   ```

4. **Create a Pull Request**

   Open a pull request against the `develop` branch of the upstream repository.

## Code Style Guidelines

### Go Code Style

We follow the standard Go style guidelines:

- Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Follow [Effective Go](https://golang.org/doc/effective_go) practices
- Use meaningful variable and function names
- Keep functions focused and reasonably sized
- Add comments explaining non-obvious code sections

Example:

```go
// BuildVFSPath constructs a complete virtual file system path
// by joining the provided components with proper separators.
func BuildVFSPath(components ...string) string {
    // Always start with a root separator
    path := "/"
    
    for _, component := range components {
        // Trim any existing separators and join
        cleanComponent := strings.Trim(component, "/")
        if cleanComponent != "" {
            path = filepath.Join(path, cleanComponent)
        }
    }
    
    return path
}
```

### PHP Code Style

For PHP code in examples and tests:

- Follow [PSR-12](https://www.php-fig.org/psr/psr-12/) coding standards
- Use 4 spaces for indentation
- Use PHP 7.2+ compatible syntax
- Always use proper error handling

Example:

```php
<?php
/**
 * Processes form data and returns a sanitized result.
 *
 * @param array $formData Raw form data
 * @return array Processed and sanitized data
 */
function processFormData(array $formData): array
{
    $result = [];
    
    foreach ($formData as $key => $value) {
        // Sanitize input to prevent XSS
        $sanitizedKey = htmlspecialchars($key, ENT_QUOTES, 'UTF-8');
        $sanitizedValue = htmlspecialchars($value, ENT_QUOTES, 'UTF-8');
        
        $result[$sanitizedKey] = $sanitizedValue;
    }
    
    return $result;
}
```

## Testing Guidelines

Frango uses Go's standard testing framework. All contributions should include appropriate tests.

### Types of Tests

1. **Unit Tests** - Test individual functions and methods
2. **Integration Tests** - Test how components work together
3. **End-to-End Tests** - Test complete functionality

### Test Coverage

- Aim for at least 80% code coverage for new features
- Always add tests for bug fixes to prevent regressions

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage report
go test -cover ./...

# Run a specific test
go test -run TestFunctionName ./path/to/package
```

### Writing Good Tests

- Each test should focus on one specific functionality
- Use descriptive test names like `TestVFSFileCreation` or `TestPHPRequestHandling`
- Use table-driven tests for testing multiple scenarios
- Include both positive and negative test cases
- Avoid test interdependencies

Example:

```go
func TestVFSFileOperations(t *testing.T) {
    tests := []struct {
        name        string
        operation   func(*VFS) error
        checkResult func(*VFS) bool
        expectError bool
    }{
        {
            name: "create file success",
            operation: func(vfs *VFS) error {
                return vfs.CreateVirtualFile("/test.txt", []byte("content"))
            },
            checkResult: func(vfs *VFS) bool {
                return vfs.FileExists("/test.txt")
            },
            expectError: false,
        },
        {
            name: "create file in non-existent directory",
            operation: func(vfs *VFS) error {
                return vfs.CreateVirtualFile("/missing/test.txt", []byte("content"))
            },
            checkResult: func(vfs *VFS) bool {
                return !vfs.FileExists("/missing/test.txt")
            },
            expectError: true,
        },
        // More test cases...
    }
    
    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            vfs := NewVFS()
            err := tc.operation(vfs)
            
            if tc.expectError && err == nil {
                t.Errorf("Expected error but got none")
            }
            
            if !tc.expectError && err != nil {
                t.Errorf("Unexpected error: %v", err)
            }
            
            if !tc.checkResult(vfs) {
                t.Errorf("Result check failed")
            }
        })
    }
}
```

## Documentation Guidelines

Good documentation is essential for Frango's usability.

### Code Comments

- Use [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc) compatible comments
- Document all exported functions, types, and constants
- Provide examples for non-obvious functionality

Example:

```go
// WithSourceDir configures the middleware to use the specified directory
// as the source for PHP files.
//
// The path should be an absolute path or relative to the current working directory.
// All PHP files in this directory will be available in the virtual filesystem.
//
// Example:
//
//     php, err := frango.New(
//         frango.WithSourceDir("./php-files"),
//     )
func WithSourceDir(dir string) Option {
    return func(m *Middleware) error {
        // Implementation...
    }
}
```

### User Documentation

- Update relevant documentation when adding features
- Write clear, concise, and accurate documentation
- Include practical examples
- Follow Markdown best practices

## Pull Request Process

1. **Ensure Tests Pass**

   All tests should pass before submitting a PR:

   ```bash
   go test ./...
   ```

2. **Update Documentation**

   Update any relevant documentation, including:
   
   - Code comments
   - User guides
   - README updates if needed

3. **Create a Pull Request**

   Submit your PR with a clear title and description:
   
   - Describe what your PR does
   - Reference any related issues using GitHub's syntax: `Fixes #123` or `Relates to #123`
   - Explain your approach and any decisions you made
   - Mention any parts you'd like specific feedback on

4. **Code Review**

   - Address all review comments
   - Make requested changes in new commits
   - Push changes to the same branch
   - Response to all comments, even if just acknowledging them

5. **PR Merge**

   Once approved, your PR will be merged by a maintainer.

## Issue Reporting Guidelines

When creating a new issue:

1. **Search First**

   Check if your issue has already been reported.

2. **Use Issue Templates**

   Follow the provided issue templates.

3. **Provide Complete Information**

   Include:
   
   - Frango version
   - Go version
   - PHP version
   - Complete steps to reproduce
   - Expected versus actual behavior
   - Code samples or test cases if possible
   - Error messages and logs

4. **Be Responsive**

   Respond to questions and requests for additional information.

## Community and Communication

### Where to Get Help

- GitHub Issues for bug reports and feature requests
- GitHub Discussions for general questions and discussions

### Communication Guidelines

- Be clear and concise
- Provide context for your questions
- Share solutions when you solve problems
- Help others when you can

## Acknowledgments

Your contributions are valued and will be recognized in the project's documentation. All contributors will be listed in the project's CONTRIBUTORS file.

Thank you for helping make Frango better! 