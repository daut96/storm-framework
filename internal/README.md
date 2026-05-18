## Explanation

The **internal folder** is designed to store various scripts, including compiled languages such as Golang, Rust, and C/C++. The **modules folder** containing compiled-language modules has been moved to `internal/source/modules`. As a result, the **modules** directory in the root only contains Python-based loader scripts.

The binary output folder from the compilation process remains located in `external/source/out`, and the storage location for Rust dependencies, namely **vendor**, remains in `external/source/dep`.
