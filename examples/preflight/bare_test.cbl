       IDENTIFICATION DIVISION.
       PROGRAM-ID. baretest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-MSG PIC X(13) VALUE "HELLO BARE!!!".
       PROCEDURE DIVISION.
       MAIN.
           DISPLAY "Bare metal works"
           DISPLAY WS-MSG
           DISPLAY 42
           STOP RUN.
