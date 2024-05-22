gcc -shared vram_dll.c -w -o vram.dll || goto end
@rem gcc vram_test3.c -o vram_test.exe && vram_test.exe
@rem withdll.exe /d:vram.dll notepad
withdll.exe /d:vram.dll ffplay D:\nadeen\valorant\audio\data\data\366738042.ogg
@rem withdll.exe /d:vram.dll "C:\Games\Sherlock Holmes - Chapter One\SH9\Binaries\Win64\SHCO.exe" -dx11
@rem withdll.exe /d:vram.dll "E:\Games\Days Gone\BendGame\Binaries\Win64\DaysGone.exe"
@rem withdll.exe /d:vram.dll python -m http.server
@rem withdll.exe /d:vram.dll %*
:end