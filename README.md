# Setup

```sh
cargo install uniffi-bindgen-go --git https://github.com/NordSecurity/uniffi-bindgen-go --tag v0.2.1+v0.25.0
apt install -y protobuf-compiler
```

Also download certs from Greenlight developer console

## Run examples

```sh
cd examples
```

Note: Examples require an already-registered node

### GetInfo

```sh
MNEMONIC="YOUR TWELVE WORD MNEMONIC HERE" cargo run --bin get-info
```

### MakeInvoice

```sh
MNEMONIC="YOUR TWELVE WORD MNEMONIC HERE" cargo run --bin make-invoice
```

## Generate bindings

```sh
 GL_CUSTOM_NOBODY_KEY=/PATH/TO/glalby/gl-certs/client-key.pem GL_CUSTOM_NOBODY_CERT=/PATH/TO/glalby/gl-certs/client.crt cargo build --release
uniffi-bindgen-go src/glalby.udl -o ffi/golang -c ./uniffi.toml
cp target/release/libglalby_bindings.so ffi/golang/glalby
```

## Run tests

```sh
cp -r ffi/golang/glalby tests/bindings/golang/
cargo test -- --nocapture
```

## Production Build

Make sure to set your gl-certs path

```sh
GL_CUSTOM_NOBODY_KEY=/PATH/TO/glalby/gl-certs/client-key.pem GL_CUSTOM_NOBODY_CERT=/PATH/TO/glalby/gl-certs/client.crt ./scripts/uniffi_bindgen_generate_go.sh
```

And then copy the outputs to `glalby-go`.

### Consume from go app

In NWC:

`go get github.com/getAlby/glalby-go`

And in the code import from `"github.com/getAlby/glalby-go/glalby"`

TODO: other platforms

## Development

1. Copy `glalby` folder into the NWC app. `cp glalby PATH/TO/NWC -r`

2. Import with `import ("github.com/getAlby/nostr-wallet-connect/glalby")`

And then you can call functions e.g. `glalby.GetInfo()`
