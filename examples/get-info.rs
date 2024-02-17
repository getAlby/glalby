use glalby_bindings::{new_blocking_greenlight_alby_client, recover};

fn main() {
    let mnemonic = std::env::var("MNEMONIC").unwrap();

    let credentials = recover(mnemonic.clone());

    let client = new_blocking_greenlight_alby_client(mnemonic, credentials);
    let result = client.get_info();

    println!("Result: {:?}", result);
}
