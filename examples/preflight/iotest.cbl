       IDENTIFICATION DIVISION.
       PROGRAM-ID. iotest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-VAL    PIC 9(5) COMP.
       PROCEDURE DIVISION.
       MAIN.
           POKE(0xB8000, 65)
           POKE(0xB8001, 7)
           PEEK(0xB8000) INTO WS-VAL
           DISPLAY WS-VAL
           PORT-OUT(0x3F8, 72)
           PORT-IN(0x3F8) INTO WS-VAL
           DISPLAY WS-VAL
           STOP RUN.
