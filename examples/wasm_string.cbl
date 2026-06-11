       IDENTIFICATION DIVISION.
       PROGRAM-ID. wasmstr.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-OUT PIC X(50).
       PROCEDURE DIVISION.
       MAIN.
           STRING "HELLO" DELIMITED BY SIZE
                  " " DELIMITED BY SIZE
                  "WORLD" DELIMITED BY SIZE
                  INTO WS-OUT
           END-STRING
           DISPLAY WS-OUT
           STOP RUN.
