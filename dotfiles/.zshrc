# Execute fish in zshrc because the nix installer adds nix to PATH after $HOME/.zprofile is sourced.
if command -v fish >/dev/null && test "$EXIT_OUT_OF_FISH" = ""; then
        exec fish
fi
