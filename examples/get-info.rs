use glalby_bindings::get_info;

fn main() {
    let phrase = std::env::var("MNEMONIC").unwrap();
    let result = get_info(phrase);
    println!("Result: {}", result)
}
