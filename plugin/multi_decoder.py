import base64
import urllib.parse
import smf

class Plugin:
    def __init__(self):
        self.name = "Advanced Dynamic Multi-Decoder"
        self.version = "1.0.0"
        
        # Registry internal untuk memetakan nama metadata ke fungsi decode yang relevan
        # Ini menjaga kode tetap clean, modular, dan sangat estetik.
        self._dispatch_table = {
            "Base64": self._decode_base64,
            "URL": self._decode_url,
            "Hex": self._decode_hex
        }

    def pre_execute(self, *args, **kwargs) -> dict:
        """
        Entry point event hook yang dipanggil oleh manager.broadcast.
        Mengevaluasi metadata secara dinamis untuk interupsi (short-circuit) atau mutasi.
        """
        metadata = kwargs.get("metadata", {})
        options = kwargs.get("options", {})
        
        # Ambil konfigurasi dari node "Transforms" di metadata
        transforms = metadata.get("Transforms", {})
        if not transforms:
            return {"handled": False}

        # Variabel penanda apakah ada data yang berhasil dimodifikasi
        mutated_options = {}
        
        # Iterasi seluruh rule yang terdaftar di tabel dispatch secara dinamis
        for transform_key, decoder_func in self._dispatch_table.items():
            # Cek apakah modul mengaktifkan flag transform ini (misal: "Base64": True)
            if transforms.get(transform_key) is True:
                smf.printd(f"[{self.name}] Match found for transform flag", transform_key, level="INFO")
                
                # Eksekusi mutasi pada data input yang ada
                for opt_key, opt_value in options.items():
                    if isinstance(opt_value, str) and opt_value.strip():
                        # Lakukan proses decode
                        decoded_value = decoder_func(opt_value)
                        
                        if decoded_value != opt_value:
                            mutated_options[opt_key] = decoded_value

        # Jika ada data yang berhasil di-decode, kembalikan ke engine untuk memutasi state
        if mutated_options:
            smf.printf(f"[✓] [{self.name}] Successfully decoded {len(mutated_options)} parameter(s).")
            
            # Skenario: Kita modifikasi options-nya, lalu izinkan engine 
            # untuk tetap fallback menjalankan current_module.execute(options_baru)
            return {
                "handled": False, 
                "modified_options": mutated_options
            }

        return {"handled": False}

    # ==========================================
    # PRIVATE DECODER STRATEGIES (Low-Level Logic)
    # ==========================================

    def _decode_base64(self, data: str) -> str:
        """Mencoba melakukan decoding Base64 secara safe."""
        try:
            # Tambahkan padding jika string base64 tidak pas (bisa merusak decode)
            padded_data = data + "=" * (-len(data) % 4)
            decoded_bytes = base64.b64decode(padded_data, validate=True)
            return decoded_bytes.decode("utf-8", errors="ignore")
        except Exception:
            return data # Fail-safe: Kembalikan data asli jika bukan b64 valid

    def _decode_url(self, data: str) -> str:
        """Melakukan URL Decoding (misal: %20 menjadi spasi, %3F menjadi ?)."""
        try:
            return urllib.parse.unquote(data)
        except Exception:
            return data

    def _decode_hex(self, data: str) -> str:
        """Mencoba melakukan decoding dari string Hexadecimal (misal: 414243 menjadi ABC)."""
        try:
            # Bersihkan prefix hex jika ada (0x atau \x)
            clean_hex = data.replace("0x", "").replace("\\x", "")
            decoded_bytes = bytes.fromhex(clean_hex)
            return decoded_bytes.decode("utf-8", errors="ignore")
        except Exception:
            return data
      
