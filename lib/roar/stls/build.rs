use std::env;
use std::path::PathBuf;

fn main() {
    let manifest_dir = PathBuf::from(env::var("CARGO_MANIFEST_DIR").unwrap());

    // Traversal dinamis ke folder external (Storm-Framework/external/...)
    let root = manifest_dir
        .parent().expect("Failed to get parent 1")
        .parent().expect("Failed to get parent 2")
        .parent().expect("Failed to get parent 3");
    
    let bssl_source_path = root.join("external/source/google/vendor/boringssl");

    // =========================================================================
    // 1. CMAKE CONFIGURATION (Hardened for FFI & Shared Objects)
    // =========================================================================
    let mut config = cmake::Config::new(&bssl_source_path);
    
    config.define("BUILD_TESTING", "OFF");
    config.define("CMAKE_BUILD_TYPE", "Release");
    
    // PERBAIKAN FATAL 1: Wajibkan PIC (Position Independent Code)
    // Ini memastikan libcrypto.a dan libssl.a aman untuk digabungkan ke dalam .so
    // dan tidak memicu MTE atau W^X violation di Android/Linux ARM64.
    config.define("CMAKE_POSITION_INDEPENDENT_CODE", "ON");

    // Optimasi BoringSSL untuk ukuran dan mematikan fitur C++ yang tidak perlu
    config.define("BUILD_SHARED_LIBS", "OFF");

    let bssl_out_dir = config.build();

    // =========================================================================
    // 2. LINKING STRATEGY
    // =========================================================================
    let build_dir = bssl_out_dir.join("build");
    let lib_dir = bssl_out_dir.join("lib");

    let potential_paths = vec![
        build_dir.join("crypto"), 
        build_dir.join("ssl"),    
        build_dir.clone(),        
        lib_dir.clone(),          
    ];

    let mut found_crypto = false;
    let mut found_ssl = false;

    for path in potential_paths {
        if path.exists() {
            if path.join("libcrypto.a").exists() || path.join("crypto.lib").exists() {
                println!("cargo:rustc-link-search=native={}", path.display());
                found_crypto = true;
            }
            if path.join("libssl.a").exists() || path.join("ssl.lib").exists() {
                println!("cargo:rustc-link-search=native={}", path.display());
                found_ssl = true;
            }
        }
    }

    if !found_crypto || !found_ssl {
        println!("cargo:rustc-link-search=native={}", build_dir.display());
    }

    println!("cargo:rustc-link-lib=static=crypto");
    println!("cargo:rustc-link-lib=static=ssl");

        // =========================================================================
    // 3. TARGET ENVIRONMENT ADJUSTMENTS (Memperbaiki Linker C++)
    // =========================================================================
    let target = env::var("TARGET").unwrap_or_default();
    
    if target.contains("android") {
        // Flag ketat untuk Android/Termux
        println!("cargo:rustc-link-arg=-Wl,--no-undefined");
        println!("cargo:rustc-link-arg=-Wl,--as-needed"); 
        println!("cargo:rustc-link-arg=-Wl,-z,relro,-z,now"); 

        // PERBAIKAN: Termux/Android NDK membutuhkan c++_shared untuk internal BoringSSL
        println!("cargo:rustc-link-lib=dylib=c++_shared");
    } else {
        // Untuk Linux murni (Debian/Ubuntu), gunakan stdc++
        println!("cargo:rustc-link-lib=dylib=stdc++");
    }

    // =========================================================================
    // 4. BINDGEN CONFIGURATION (Strict C-ABI)
    // =========================================================================
    let header_path = bssl_source_path.join("include/openssl/ssl.h");
    let include_path = bssl_source_path.join("include");
    
    // Rerun build script HANYA jika header SSL atau file build ini berubah
    println!("cargo:rerun-if-changed={}", header_path.display());
    println!("cargo:rerun-if-changed=build.rs");

    let bindings = bindgen::Builder::default()
        .header(header_path.to_str().unwrap())
        .clang_arg(format!("-I{}", include_path.display()))
        .allowlist_function("TLS_.*")
        .allowlist_function("SSL_.*")
        .allowlist_type("SSL_.*")
        .allowlist_var("SSL_.*")
        .allowlist_var("TLS1_.*")
        // PERBAIKAN 3: Gunakan tipe core Rust (ctypes) yang lebih ketat
        .use_core() 
        .clang_arg("-D__STDC_CONSTANT_MACROS")
        .clang_arg("-D__STDC_FORMAT_MACROS")
        .clang_arg("-D__STDC_LIMIT_MACROS")
        .generate()
        .expect("Failed to translate BoringSSL header");

    let out_path = PathBuf::from(env::var("OUT_DIR").unwrap());
    bindings
        .write_to_file(out_path.join("bssl_bindings.rs"))
        .expect("Failed to write FFI bindings file");
}
