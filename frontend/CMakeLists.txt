find_package(Qt6 6.5 REQUIRED COMPONENTS Widgets)

qt_add_executable(NoteHub WIN32 MACOSX_BUNDLE
    src/main.cpp
    src/IpcClient.cpp
    src/IpcClient.h
    src/Theme.cpp
    src/Theme.h
    src/MemoCard.cpp
    src/MemoCard.h
    src/MemoDialog.cpp
    src/MemoDialog.h
    src/ShareDialog.cpp
    src/ShareDialog.h
    src/MainWindow.cpp
    src/MainWindow.h
    resources/notehub.qrc
)

if(WIN32)
    target_sources(NoteHub PRIVATE resources/notehub.rc)
endif()

target_link_libraries(NoteHub PRIVATE Qt6::Widgets)
target_compile_definitions(NoteHub PRIVATE NOTEHUB_VERSION="${PROJECT_VERSION}")

if(MSVC)
    target_compile_options(NoteHub PRIVATE /W4 /permissive- /utf-8)
else()
    target_compile_options(NoteHub PRIVATE -Wall -Wextra -Wpedantic)
endif()

set_target_properties(NoteHub PROPERTIES
    OUTPUT_NAME "NoteHub"
    MACOSX_BUNDLE_BUNDLE_NAME "NoteHub"
    MACOSX_BUNDLE_GUI_IDENTIFIER "io.notehub.desktop"
    MACOSX_BUNDLE_SHORT_VERSION_STRING "${PROJECT_VERSION}"
    MACOSX_BUNDLE_BUNDLE_VERSION "${PROJECT_VERSION}"
)

install(TARGETS NoteHub
    BUNDLE DESTINATION .
    RUNTIME DESTINATION .
)
