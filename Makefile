BIN=$(DESTDIR)/usr/bin
TARGET=newt
build:
	./build.sh

install:
	install -d $(BIN)
	install $(TARGET)/$(TARGET) $(BIN)
	rm -f  $(TARGET)/$(TARGET)
