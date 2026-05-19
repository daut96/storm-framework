from typing import Any


class RuntimeContext:
    def __init__(self, metadata: dict, plugin_manager: Any):
        self.metadata = metadata
        self._plugin = plugin_manager
        self.packet_count = 0  # Modul bisa menyimpan state runtime di sini jika mau

    def transform(self, payload: str) -> str:
        """
        Jembatan pencit-pencit (interceptor) payload ke plugin.
        Modul cukup panggil: runtime.transform(payload)
        """
        if "Transforms" not in self.metadata:
            return payload

        # Lempar ke mesin broadcast yang sudah kita perbaiki kemarin
        results = self._plugin.broadcast(
            "on_payload_ready", payload=payload, metadata=self.metadata
        )

        current_payload = payload
        if isinstance(results, dict):
            for plugin_name, res in results.items():
                if isinstance(res, dict) and "mutated_payload" in res:
                    current_payload = res["mutated_payload"]

        return current_payload


runtime = RuntimeContext()
