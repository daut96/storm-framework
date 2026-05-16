# -- https://github.com/StormWorld0/storm-framework
# -- SMF License
import smf

from apps.utility.verify import *
from lib.roar.plugin_api import plugin
from lib.roar.cache import cache_modules as cache
from lib.roar.callbin import calling


def boot():
    smf.printd("Boot starting...", level="INFO")

    # Check core startup security
    check_critical_files()
    smf.printd("Check core success", level="INFO")

    # Boot Plugin Manager
    plugin.boot()
    smf.printd("Boot plugin successfuls", level="INFO")

    # Cache modules synchronization
    cache.sync_modules()
    smf.printd("Module synchronization successful", level="INFO")

    # Cache binary synchronization
    calling.sync_bin()
    smf.printd("Binary synchronization successful", level="INFO")

    # Verify file integrity
    run_verif()
    smf.printd("Verification of integrity check success", level="INFO")
