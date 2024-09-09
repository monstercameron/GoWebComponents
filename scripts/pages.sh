#!/bin/bash

# Function to check if the previous command failed
check_error() {
    if [ $? -ne 0 ]; then
        echo "Error occurred: $1"
        exit 1
    fi
}

# Check for local gh-pages branch and delete it if it exists
if git show-ref --quiet refs/heads/gh-pages; then
    echo "Local gh-pages branch found. Deleting it..."
    git branch -D gh-pages
    check_error "Failed to delete local gh-pages branch."
else
    echo "No local gh-pages branch found. Continuing..."
fi

# Build the WebAssembly project with maximum optimizations
echo "Building WebAssembly with optimizations..."
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o static/bin/main.wasm
check_error "Failed to build WebAssembly project."

# Commit the latest changes with the message "gh-pages build"
echo "Committing latest changes..."
git add .
git commit -m "gh-pages build"
check_error "Failed to commit changes."

# Checkout a new clean gh-pages branch
echo "Switching to a new gh-pages branch..."
git checkout -b gh-pages
check_error "Failed to create and checkout new gh-pages branch."

# Merge the master (main) branch into gh-pages, allowing unrelated histories for orphaned branch
echo "Merging master into gh-pages..."
git merge master --allow-unrelated-histories --no-edit
check_error "Failed to merge master into gh-pages."

# Remove all files and folders except static and .git
echo "Removing old files, keeping static and .git..."
find . -mindepth 1 -not -path "./static*" -not -path "./.git*" -delete
check_error "Failed to clean up files."

# Move the contents of the static folder to the root
echo "Moving static content to root..."
mv static/* ./
check_error "Failed to move static content to root."

# Delete the static folder
echo "Deleting static folder..."
rm -rf static
check_error "Failed to delete static folder."

# Add the changes and commit them
echo "Committing changes to gh-pages..."
git add .
git commit -m "Deploy updated files to gh-pages"
check_error "Failed to commit changes to gh-pages."

# Push changes to the GitHub repository
echo "Pushing changes to GitHub..."
git push -f origin gh-pages
check_error "Failed to push changes to GitHub."

echo "Build, deployment, and push to GitHub complete."
