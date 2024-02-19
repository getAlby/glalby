use glalby_bindings::{new_blocking_greenlight_alby_client, recover, MakeInvoiceRequest};

fn main() {
    let mnemonic = std::env::var("MNEMONIC").unwrap();

    let credentials = recover(mnemonic.clone()).unwrap();

    let client = new_blocking_greenlight_alby_client(mnemonic, credentials).unwrap();
    let result = client
        .make_invoice(MakeInvoiceRequest {
            amount_msat: 1000,
            description: String::from("Test description"),
            label: rand::random::<u64>().to_string(),
        })
        .unwrap();

    println!("Result: {}", result.bolt11);
}
