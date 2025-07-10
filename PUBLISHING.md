# Publishing Guide for interop-mcp-server

This guide explains how to publish the `interop-mcp-server` npm package.

## Prerequisites

1. **npm account**: Create an account at [npmjs.com](https://www.npmjs.com/)
2. **npm CLI**: Install with `npm install -g npm`
3. **Authentication**: Run `npm login` to authenticate

## Pre-Publishing Checklist

1. **Test the package locally**:
   ```bash
   npm test
   node bin/interop-mcp-server.js --test
   ```

2. **Update version** in `package.json` (follow semantic versioning):
   ```bash
   npm version patch  # for bug fixes
   npm version minor  # for new features
   npm version major  # for breaking changes
   ```

3. **Verify package contents**:
   ```bash
   npm pack --dry-run
   ```

## Publishing Steps

### 1. Initial Publication

```bash
# Make sure you're in the project root
cd /path/to/interop

# Run tests
npm test

# Publish to npm
npm publish
```

### 2. Subsequent Updates

```bash
# Update version
npm version patch

# Publish
npm publish
```

### 3. Publishing with Tags

For pre-releases or beta versions:

```bash
# Publish as beta
npm publish --tag beta

# Publish as next
npm publish --tag next
```

## Post-Publishing

1. **Verify the package**:
   ```bash
   npx interop-mcp-server --version
   ```

2. **Test installation**:
   ```bash
   # In a temporary directory
   mkdir test-install && cd test-install
   npx interop-mcp-server --help
   ```

3. **Update documentation** if needed

## Package Structure

The published package will include:

```
interop-mcp-server/
├── package.json
├── README.md
├── bin/
│   └── interop-mcp-server.js
├── lib/
│   ├── index.js
│   └── platform.js
└── scripts/
    ├── install-binary.js
    └── prepare.js
```

## Version Management

- **Patch** (x.y.Z): Bug fixes, security updates
- **Minor** (x.Y.z): New features, backward compatible
- **Major** (X.y.z): Breaking changes

## Distribution Strategy

1. **npm registry**: Primary distribution method
2. **GitHub releases**: Link to npm package
3. **Documentation**: Update main project README

## Maintenance

- **Monitor downloads**: Check npm stats
- **Update dependencies**: Regular security updates
- **Sync with main project**: Keep in sync with Interop releases
- **User feedback**: Monitor issues and feature requests

## Troubleshooting

### Common Issues

1. **Authentication failed**:
   ```bash
   npm logout
   npm login
   ```

2. **Package name taken**:
   - Update `name` in `package.json`
   - Check availability: `npm view package-name`

3. **Version already exists**:
   ```bash
   npm version patch
   npm publish
   ```

### Testing Before Publishing

```bash
# Test package locally
npm link
npx interop-mcp-server --test

# Test in clean environment
docker run -it --rm node:18 bash
npx interop-mcp-server --help
```

## Support

- **Issues**: [GitHub Issues](https://github.com/yigitozgumus/interop/issues)
- **npm**: [Package Page](https://www.npmjs.com/package/interop-mcp-server)
- **Documentation**: Main project README 