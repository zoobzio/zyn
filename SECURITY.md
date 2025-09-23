# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

If you discover a security vulnerability in zyn, please follow these steps:

1. **DO NOT** open a public issue
2. Email security concerns to: security@zoobzio.com
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

## Security Considerations

### API Keys

- Never commit API keys to version control
- Use environment variables for credentials
- Rotate keys regularly
- Use separate keys for development/production

### Provider Security

```go
// Good - API key from environment
provider := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
})

// Bad - Hardcoded API key
provider := openai.New(openai.Config{
    APIKey: "sk-1234567890abcdef", // NEVER DO THIS
})
```

### Prompt Injection

Be aware of prompt injection risks when using user input:

```go
// Validate and sanitize user input
userInput := sanitize(request.Input)

// Use structured prompts to reduce injection risk
result, err := synapse.Fire(ctx, userInput)
```

### Rate Limiting

Protect your API keys from abuse:

```go
synapse := Binary("question", provider,
    WithRateLimit(10, 100), // 10 requests/sec, burst of 100
    WithCircuitBreaker(5, 30*time.Second),
)
```

## Security Updates

Security updates are released as soon as possible after discovery and verification. Update to the latest version promptly.

## Acknowledgments

We appreciate responsible disclosure of security vulnerabilities. Contributors who report valid security issues will be acknowledged (unless they prefer to remain anonymous).