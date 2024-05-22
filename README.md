# vram - **Download More RAM**

VRAM allows you to share memory over a network in Windows. Tested with multiple games.

Use at your own risk. This may break your computer (I'm not responsible).

**DO NOT USE** with games that have anti-cheat systems.

Don't ask me about naming or coding conventions :)

### Requirements
- Windows 10 64-bit (maybe win11?)
- Go
- GCC
- [withdll.exe](https://github.com/microsoft/Detours/blob/main/samples/withdll/withdll.cpp) from [microsoft/Detours](https://github.com/microsoft/Detours)
- Maybe [WinFsp](https://github.com/winfsp/winfsp)?

## How to Use

### Remote Setup:
1. Run `vram3_remote_memfs/build_server.cmd` to build the server executable.
2. Open tcp port 7885 on remote computer
3. Execute the resulting `vram_server.exe` on the remote computer (can be on Linux).

### Local Setup:
1. Update remote_addr to remote ip/addr in `vram3_remote_memfs/data_remote.go`
2. Start vram filesystem: `vram3_remote_memfs/runfs.cmd`
3. Start vram vm server: `vram3/runvm.cmd`
4. Build `vram.dll`: `vram3/build.cmd`
5. Start the target process:
    ```sh
    withdll.exe /d:vram.dll <cmd> <args>
    ```

### TODO
- Remove hardcoded Z:/ drive letter
- Add compression
- Support multiple remotes
- Fix bugs?

2023 Nadeen Udantha <me@nadeen.lk>. All rights reserved.

`memfs.go` and `trace.go` © 2017-2022 [Bill Zissimopoulos](https://github.com/billziss-gh).