# Setup

```sh
cargo install uniffi-bindgen-go --git https://github.com/NordSecurity/uniffi-bindgen-go --tag v0.2.1+v0.25.0
apt install -y protobuf-compiler
```

Also download certs from Greenlight developer console

## Run as a temporary app (just testing things)

```sh
MNEMONIC="YOUR TWELVE WORD MNEMONIC HERE" GL_CUSTOM_NOBODY_KEY=/PATH/TO/glalby/gl-certs/client-key.pem GL_CUSTOM_NOBODY_CERT=/PATH/TO/glalby/gl-certs/client.crt cargo run --bin glalby-temp
```

## Generate bindings

```sh
cargo build --release --bin uniffi-bindgen
uniffi-bindgen-go src/glalby.udl -o ffi/golang -c ./uniffi.toml
cp target/release/libglalby_bindings.so ffi/golang
cp -r ffi/golang/glalby tests/bindings/golang/
cargo test -- --nocapture
```

## Consume from go app

Copy `libglalby_bindings.so` into `glalby` folder and then copy `glalby` folder into NWC app.

Import with `import ("github.com/getAlby/nostr-wallet-connect/glalby")`

```sh
CGO_LDFLAGS="-lglalby_bindings -L./glalby -Wl,-rpath,./glalby" go run .
```
