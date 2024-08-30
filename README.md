
# Filebrowser-Upload

A CLI utility to upload files to [filebrowser](https://github.com/filebrowser/filebrowser)


## Installation

Binary executables available in [releases](https://github.com/spotdemo4/filebrowser-upload/releases)

## Nix Installation

Add the repository to your flake inputs
```nix
inputs = {
    ...
    filebrowser-upload.url = "github:spotdemo4/filebrowser-upload";
};
```
Add the overlay to nixpkgs
```nix
nixpkgs = {
    ...
    overlays = [
        ...
        inputs.filebrowser-upload.overlays.default
    ];
};
```
Finally, add filebrowser-upload to your packages
```nix
environment.systemPackages = with pkgs; [
    ...
    filebrowser-upload
];
```

## Usage/Examples

```
filebrowser-upload set \
    -url https://filebrowser.example.com \
    -username trev \
    -password 123 \
    -override \
    -share
```
```
filebrowser-upload upload \
    -file kitten.png \
    -directory cutekittens
```
```
filebrowser-upload upload \
    -file kitten.png \
    -directory cutekittens \
    -url https://filebrowser.example.com \
    -username trev \
    -password 123 \
    -override \
    -share
```
