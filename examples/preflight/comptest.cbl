       IDENTIFICATION DIVISION.
       PROGRAM-ID. comptest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-A            PIC 9(5) COMP VALUE 17.
       01 WS-B            PIC 9(5) COMP VALUE 5.
       01 WS-C            PIC 9(5) COMP VALUE 0.
       PROCEDURE DIVISION.
       MAIN.
           COMPUTE WS-C = WS-A / WS-B
           IF WS-C = 3
               DISPLAY 1
           ELSE
               DISPLAY WS-C
           END-IF
           COMPUTE WS-C = (WS-A + WS-B) * 2
           IF WS-C = 44
               DISPLAY 1
           ELSE
               DISPLAY WS-C
           END-IF
           STOP RUN.
