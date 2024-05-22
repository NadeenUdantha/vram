#include <windows.h>
#include <stdint.h>
#include <stdio.h>
#include <psapi.h>

__declspec(dllexport) void dummy() {}

char *lasterr()
{
    char *bb = 0;
    FormatMessageA(
        FORMAT_MESSAGE_FROM_SYSTEM | FORMAT_MESSAGE_IGNORE_INSERTS | FORMAT_MESSAGE_ALLOCATE_BUFFER,
        0, GetLastError(), MAKELANGID(LANG_NEUTRAL, SUBLANG_DEFAULT), &bb, 0, 0);
    return bb;
}

void _assert(int x, char *file, int line)
{
    if (!x)
    {
        char *b = malloc(1024);
        b[snprintf(b, 1024, "@%s:%d\nWTF???\nLastError=%s\n", file, line, lasterr())] = 0;
        MessageBoxA(GetConsoleWindow(), b, "UwU", MB_OK | MB_ICONERROR);
        ExitProcess(-1);
    }
}

#define assert(x) _assert(x, __FILE__, __LINE__)

#define logf(fmt, ...)                                                          \
    {                                                                           \
        char *b = malloc(1024);                                                 \
        b[snprintf(b, 1024, fmt, ##__VA_ARGS__)] = 0;                           \
        /*MessageBoxA(GetForegroundWindow(), b, "UwU", MB_OK | MB_ICONERROR);*/ \
        WriteFile(GetStdHandle(STD_OUTPUT_HANDLE), b, strlen(b), 0, 0);         \
        free(b);                                                                \
        /*FlushFileBuffers(GetStdHandle(STD_OUTPUT_HANDLE));*/                  \
    }

#ifdef _MSVC_TRADITIONAL
#define logf(fmt, ...)
#endif

#define logf(fmt, ...)

typedef struct
{
    uint64_t retval;
    uint64_t addr;
    uint64_t size;
    uint32_t type;
    uint32_t protect;
    uint32_t err;
    char method[32];
} vram_msg;

HANDLE pipe;
vram_msg *vram_tx(char *method, uint64_t retval, uint64_t addr, uint64_t size, uint32_t type, uint32_t protect, uint32_t err)
{
    vram_msg *a = malloc(sizeof(vram_msg));
    strcpy(a->method, method);
    a->retval = retval;
    a->addr = addr;
    a->size = size;
    a->type = type;
    a->protect = protect;
    a->err = err;
    vram_msg *b = malloc(sizeof(vram_msg));
    DWORD x;
    assert(TransactNamedPipe(pipe, a, sizeof(vram_msg), b, sizeof(vram_msg), &x, 0));
    assert(x == sizeof(vram_msg));
    free(a);
    return b;
}

FARPROC zVirtualAlloc;
LPVOID WINAPI xVirtualAlloc(uint64_t lpAddress, SIZE_T dwSize, DWORD flAllocationType, DWORD flProtect)
{
    // flAllocationType &= ~MEM_TOP_DOWN;
    // return zVirtualAlloc(lpAddress, dwSize, flAllocationType, flProtect);
    vram_msg *msg = vram_tx("VirtualAlloc", 0, lpAddress, dwSize, flAllocationType, flProtect, 0);
    uint64_t x = msg->retval;
    SetLastError(msg->err);
    free(msg);
    return x;
}

FARPROC zVirtualProtect = VirtualProtect;
BOOL WINAPI xVirtualProtect(LPVOID lpAddress, SIZE_T dwSize, DWORD flNewProtect, PDWORD lpflOldProtect)
{
    // return zVirtualProtect(lpAddress, dwSize, flNewProtect, lpflOldProtect);
    vram_msg *msg = vram_tx("VirtualProtect", 0, lpAddress, dwSize, 0, flNewProtect, 0);
    if (lpflOldProtect)
        *lpflOldProtect = msg->protect;
    BOOL x = msg->retval;
    SetLastError(msg->err);
    free(msg);
    return x;
}

FARPROC zVirtualFree;
BOOL WINAPI xVirtualFree(LPVOID lpAddress, SIZE_T dwSize, DWORD dwFreeType)
{
    // return zVirtualFree(lpAddress, dwSize, dwFreeType);
    vram_msg *msg = vram_tx("VirtualFree", 0, lpAddress, dwSize, dwFreeType, 0, 0);
    BOOL x = msg->retval;
    SetLastError(msg->err);
    free(msg);
    return x;
}

FARPROC zVirtualQuery;
SIZE_T WINAPI xVirtualQuery(LPCVOID lpAddress, PMEMORY_BASIC_INFORMATION64 lpBuffer, SIZE_T dwLength)
{
    assert(lpBuffer != 0 && dwLength == sizeof(MEMORY_BASIC_INFORMATION64));
    // return zVirtualQuery(lpAddress, lpBuffer, dwLength);
    vram_msg *msg = vram_tx("VirtualQuery", lpBuffer, lpAddress, 0, 0, 0, 0);
    SIZE_T x = msg->retval;
    SetLastError(msg->err);
    free(msg);
    return x;
}

FARPROC zVirtualLock;
BOOL WINAPI xVirtualLock(LPVOID lpAddress, SIZE_T dwSize)
{
    // return zVirtualLock(lpAddress, dwSize);
    return 1;
    /*vram_msg *msg = vram_tx("VirtualLock", 0, lpAddress, 0, dwSize, 0, 0, 0, 0, 0);
    BOOL x = msg->addr;
    SetLastError(msg->err);
    free(msg);
    return x;*/
}

FARPROC zVirtualUnlock;
BOOL WINAPI xVirtualUnlock(LPVOID lpAddress, SIZE_T dwSize)
{
    // return zVirtualUnlock(lpAddress, dwSize);
    return 1;
    /*vram_msg *msg = vram_tx("VirtualUnlock", 0, lpAddress, 0, dwSize, 0, 0, 0, 0, 0);
    BOOL x = msg->addr;
    SetLastError(msg->err);
    free(msg);
    return x;*/
}

void hookx(char *name, FARPROC proc, FARPROC hook, FARPROC *real)
{
    logf("hook(%s,%llx,%llx)\n", name, proc, hook);
    uint8_t *x = proc;
    uint32_t p;
    // if (x[0] == 0xff && x[1] == 0x25 && *(uint32_t *)(x + 2) == 0)return;
    if (!(x[0] == 0x48 && x[1] == 0xff & x[2] == 0x25))
    {
        for (int z = 0; z < 16; z++)
            printf("%02x ", x[z]);
        puts("?");
        assert(0);
    }
    assert(*(uint64_t *)(x + 7) == 0xcccccccccccccccc);
    if (real)
        *real = *(uint64_t *)(x + 3 + 4 + *(int32_t *)(x + 3));
    assert(zVirtualProtect(proc, 14, PAGE_EXECUTE_READWRITE, &p));
    *x++ = 0xff;
    *x++ = 0x25;
    *(uint32_t *)x = 0;
    x += 4;
    *(uint64_t *)x = hook;
    assert(zVirtualProtect(proc, 14, p, &p));
    assert(FlushInstructionCache(GetCurrentProcess(), proc, 14));
}

#define hook(x, y, z) hookx(#x, x, y, z)

void block_fn()
{
    assert(0);
}

void blockx(FARPROC proc, char *name)
{
    printf("block(%s,%llx)\n", name, proc);
    if (proc)
        hookx(name, proc, block_fn, 0);
}

#define block(x) blockx(x, #x)
//#define block2(x) blockx(GetProcAddress(LoadLibraryA("kernel32.dll"), #x), #x)

void writeHandle(HANDLE pipe)
{
    HANDLE ph = OpenProcess(PROCESS_VM_OPERATION | PROCESS_QUERY_INFORMATION | PROCESS_VM_WRITE | PROCESS_TERMINATE, 0, GetCurrentProcessId());
    assert(ph);
    DWORD rpid;
    assert(GetNamedPipeServerProcessId(pipe, &rpid));
    HANDLE rph = OpenProcess(PROCESS_DUP_HANDLE, 0, rpid);
    assert(rph);
    uint64_t dph;
    assert(DuplicateHandle(GetCurrentProcess(), ph, rph, &dph, 0, 0, DUPLICATE_SAME_ACCESS));
    logf("ph=%llx rph=%llx dph=%llx\n", ph, rph, dph);
    free(vram_tx("ph", dph, 0, 0, 0, 0, 0));
    assert(CloseHandle(ph));
    assert(CloseHandle(rph));
}

BOOL WINAPI DllMain(HINSTANCE hinst, DWORD dwReason, LPVOID reserved)
{
    if (dwReason == DLL_PROCESS_ATTACH)
    {
        AllocConsole();
        logf("vram attached\n");
        {
            pipe = CreateFileA("\\\\.\\pipe\\vramKNUS", GENERIC_READ | GENERIC_WRITE, 0, 0, OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, 0);
            assert(pipe != INVALID_HANDLE_VALUE);
            uint32_t x = PIPE_READMODE_MESSAGE;
            assert(SetNamedPipeHandleState(pipe, &x, 0, 0));
            writeHandle(pipe);
        }
        block(AllocateUserPhysicalPages);
        block(FlushViewOfFile);
        block(FreeUserPhysicalPages);
        block(GetWriteWatch);
        block(MapUserPhysicalPages);
        block(ResetWriteWatch);
        block(VirtualAllocEx);
        block(VirtualFreeEx);
        block(VirtualProtectEx);
        block(VirtualQueryEx);
        hook(VirtualAlloc, xVirtualAlloc, &zVirtualAlloc);
        hook(VirtualProtect, xVirtualProtect, &zVirtualProtect);
        hook(VirtualFree, xVirtualFree, &zVirtualFree);
        hook(VirtualQuery, xVirtualQuery, &zVirtualQuery);
        hook(VirtualLock, xVirtualLock, &zVirtualLock);
        hook(VirtualUnlock, xVirtualUnlock, &zVirtualUnlock);
    }
    return TRUE;
}
