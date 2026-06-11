       IDENTIFICATION DIVISION.
       PROGRAM-ID. typetest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-NUM PIC 9(4).
       01 WS-TXT PIC X(10).
       PROCEDURE DIVISION.
       MAIN.
           COMPUTE WS-TXT = 5 + 3
           COMPUTE WS-NUM = WS-TXT + 1
           STOP RUN.
