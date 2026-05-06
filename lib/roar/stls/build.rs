use std::env;
use std::path::PathBuf;

fn main() {
    // 1. Dapatkan lokasi folder 'stls' saat ini (di mana Cargo.toml berada)
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());

    // 2. Traversal dinamis ke folder external (Storm-Framework/external/...)
    // Mundur 3 kali: stls -> roar -> lib -> Storm-Framework
    let root = manifest_dir
        .parent().unwrap()
        .parent().unwrap()
        .parent().unwrap();
    
    let bssl_source_path = root.join("external/source/google/vendor/boringssl");

    // 3. Instruksi CMake (Dijalankan di sisi pengguna)
    let mut config = cmake::Config::new(&bssl_source_path);
    
    // Optimasi kompilasi untuk user (Release mode & tanpa testing)
    config.define("BUILD_TESTING", "OFF");
    config.define("CMAKE_BUILD_TYPE", "Release");

    // Jika user di Android (Termux), CMake akan menyesuaikan toolchain secara otomatis
    let bssl_out_dir = config.build();

    // 4. LINKING: Menghubungkan library internal ke hasil kompilasi BoringSSL
    // Kita arahkan linker pengguna untuk mencari file .a di folder output
    println!("cargo:rustc-link-search=native={}/build/crypto", bssl_out_dir.display());
    println!("cargo:rustc-link-search=native={}/build/ssl", bssl_out_dir.display());

    // Menggabungkan secara statis agar hasil .so milik user mandiri (standalone)
    println!("cargo:rustc-link-lib=static=crypto");
    println!("cargo:rustc-link-lib=static=ssl");

    // 5. BINDGEN: Menghasilkan peta fungsi untuk Rust pengguna
    let header_path = bssl_source_path.join("include/openssl/ssl.h");
    let include_path = bssl_source_path.join("include");
    let bindings = bindgen::Builder::default()
        .header(header_path.to_str().unwrap())
        .clang_arg(format!("-I{}", include_path.display())) 
        .allowlist_function("SSL_.*")
        .allowlist_type("SSL_.*")
        .allowlist_var("SSL_.*")
        .clang_arg("-D__STDC_CONSTANT_MACROS")
        .clang_arg("-D__STDC_FORMAT_MACROS")
        .clang_arg("-D__STDC_LIMIT_MACROS")
        .generate()
        .expect("Failed to translate BoringSSL header on user device");

    let out_path = PathBuf::from(env::var("OUT_DIR").unwrap());
    bindings
        .write_to_file(out_path.join("bssl_bindings.rs"))
        .expect("Failed to write FFI bindings file");
}

