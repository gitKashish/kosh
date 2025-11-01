#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
BUILD_DIR="bin"
APP_NAME="kosh"
SOURCE_DIR="./src"

# Print usage
usage() {
    echo -e "${BLUE}Usage:${NC} $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -a, --all              Build for all platforms (windows, linux, darwin)"
    echo "  -p, --platform OS      Build for specific platform (windows|linux|darwin|all)"
    echo "  -d, --debug            Build debug version only"
    echo "  -r, --release          Build release version only"
    echo "  -c, --clean            Clean build directory before building"
    echo "  -h, --help             Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                           # Build both debug and release for current platform"
    echo "  $0 -p windows                # Build for Windows only"
    echo "  $0 -a                        # Build for all platforms"
    echo "  $0 -p linux -d               # Build debug version for Linux"
    echo "  $0 --clean --all             # Clean and build for all platforms"
    exit 0
}

# Clean build directory
clean_build_dir() {
    if [ -d "$BUILD_DIR" ]; then
        echo -e "${YELLOW}Cleaning build directory...${NC}"
        rm -rf "$BUILD_DIR"
    fi
    mkdir -p "$BUILD_DIR"
}

# Build function
build() {
    local os=$1
    local arch=$2
    local build_type=$3
    local output_name=$4
    local build_tags=$5

    echo -e "${BLUE}Building ${build_type} for ${os}/${arch}...${NC}"

    # Set environment variables
    export GOOS=$os
    export GOARCH=$arch

    # Build command
    if [ -n "$build_tags" ]; then
        go build -tags "$build_tags" -ldflags="-s -w" -o "${BUILD_DIR}/${output_name}" $SOURCE_DIR
    else
        go build -ldflags="-s -w" -o "${BUILD_DIR}/${output_name}" $SOURCE_DIR
    fi

    if [ $? -eq 0 ]; then
        local size=$(du -h "${BUILD_DIR}/${output_name}" | cut -f1)
        echo -e "${GREEN}✓ Successfully built: ${BUILD_DIR}/${output_name} (${size})${NC}"
        return 0
    else
        echo -e "${RED}✗ Build failed for ${os}/${arch} ${build_type}${NC}"
        return 1
    fi
}

# Build for specific platform
build_platform() {
    local os=$1
    local build_debug=$2
    local build_release=$3

    case $os in
        windows)
            if [ "$build_release" = true ]; then
                build "windows" "amd64" "release" "${APP_NAME}-windows-amd64.exe" ""
            fi
            if [ "$build_debug" = true ]; then
                build "windows" "amd64" "debug" "${APP_NAME}-windows-amd64-debug.exe" "debug"
            fi
            ;;
        linux)
            if [ "$build_release" = true ]; then
                build "linux" "amd64" "release" "${APP_NAME}-linux-amd64" ""
                build "linux" "arm64" "release" "${APP_NAME}-linux-arm64" ""
            fi
            if [ "$build_debug" = true ]; then
                build "linux" "amd64" "debug" "${APP_NAME}-linux-amd64-debug" "debug"
                build "linux" "arm64" "debug" "${APP_NAME}-linux-arm64-debug" "debug"
            fi
            ;;
        darwin)
            if [ "$build_release" = true ]; then
                build "darwin" "amd64" "release" "${APP_NAME}-darwin-amd64" ""
                build "darwin" "arm64" "release" "${APP_NAME}-darwin-arm64" ""
            fi
            if [ "$build_debug" = true ]; then
                build "darwin" "amd64" "debug" "${APP_NAME}-darwin-amd64-debug" "debug"
                build "darwin" "arm64" "debug" "${APP_NAME}-darwin-arm64-debug" "debug"
            fi
            ;;
        *)
            echo -e "${RED}Unknown platform: $os${NC}"
            return 1
            ;;
    esac
}

# Detect current platform
detect_platform() {
    case "$(uname -s)" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        MINGW*|MSYS*|CYGWIN*) echo "windows";;
        *)          echo "unknown";;
    esac
}

# Generate checksums
generate_checksums() {
    echo -e "${BLUE}Generating checksums...${NC}"
    cd "$BUILD_DIR"
    
    if command -v sha256sum &> /dev/null; then
        sha256sum * > checksums.txt
        echo -e "${GREEN}✓ Checksums saved to ${BUILD_DIR}/checksums.txt${NC}"
    else
        echo -e "${YELLOW}⚠ sha256sum not found, skipping checksums${NC}"
    fi
    
    cd - > /dev/null
}

# Create release archive
create_archives() {
    echo -e "${BLUE}Creating release archives...${NC}"
    cd "$BUILD_DIR"
    
    for file in *; do
        if [ -f "$file" ] && [ "$file" != "checksums.txt" ] && [[ ! "$file" =~ \.zip$ ]] && [[ ! "$file" =~ \.tar\.gz$ ]]; then
            if [[ "$file" == *.exe ]]; then
                # Windows - use zip
                zip -q "${file%.exe}.zip" "$file"
                echo -e "${GREEN}✓ Created ${file%.exe}.zip${NC}"
            else
                # Unix - use tar.gz
                tar -czf "${file}.tar.gz" "$file"
                echo -e "${GREEN}✓ Created ${file}.tar.gz${NC}"
            fi
        fi
    done
    
    cd - > /dev/null
}

# Main script
main() {
    local platform=""
    local build_debug=true
    local build_release=true
    local clean=false
    local build_all=false

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -c|--clean)
                clean=true
                shift
                ;;
            -a|--all)
                build_all=true
                shift
                ;;
            -p|--platform)
                platform="$2"
                shift 2
                ;;
            -d|--debug)
                build_release=false
                shift
                ;;
            -r|--release)
                build_debug=false
                shift
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                usage
                ;;
        esac
    done

    # Clean if requested
    if [ "$clean" = true ]; then
        clean_build_dir
    else
        mkdir -p "$BUILD_DIR"
    fi

    # Determine platforms to build
    local platforms=()
    
    if [ "$build_all" = true ] || [ "$platform" = "all" ]; then
        platforms=("windows" "linux" "darwin")
    elif [ -n "$platform" ]; then
        platforms=("$platform")
    else
        # Build for current platform only
        current_platform=$(detect_platform)
        if [ "$current_platform" = "unknown" ]; then
            echo -e "${RED}Unable to detect platform. Please specify with -p${NC}"
            exit 1
        fi
        platforms=("$current_platform")
    fi

    # Build for each platform
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  Building ${APP_NAME}                        ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
    echo ""

    local success=0
    local failed=0

    for plat in "${platforms[@]}"; do
        build_platform "$plat" "$build_debug" "$build_release"
        if [ $? -eq 0 ]; then
            ((success++))
        else
            ((failed++))
        fi
    done

    generate_checksums
    create_archives

    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  Build Summary                         ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
    echo -e "${GREEN}Successful: $success${NC}"
    if [ $failed -gt 0 ]; then
        echo -e "${RED}Failed: $failed${NC}"
    fi
    echo ""
    echo -e "${BLUE}Output directory: ${BUILD_DIR}${NC}"
    
    # List built files
    if [ -d "$BUILD_DIR" ] && [ "$(ls -A $BUILD_DIR)" ]; then
        echo ""
        echo -e "${BLUE}Built files:${NC}"
        ls -lh "$BUILD_DIR" | tail -n +2 | awk '{printf "  %s  %s\n", $5, $9}'
    fi
}

# Run main function
main "$@"