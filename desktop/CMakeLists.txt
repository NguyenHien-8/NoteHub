add_library(notehub-ui STATIC
    src/BackendClient.cpp src/BackendClient.h
    src/NoteEditor.cpp src/NoteEditor.h
    src/ImageGallery.cpp src/ImageGallery.h
    src/MainWindow.cpp src/MainWindow.h)
target_link_libraries(notehub-ui PUBLIC Qt6::Widgets Qt6::Concurrent)
target_include_directories(notehub-ui PUBLIC src)
target_compile_definitions(notehub-ui PUBLIC NOTEHUB_VERSION="${PROJECT_VERSION}")
if(MSVC)
    target_compile_options(notehub-ui PRIVATE /W4 /utf-8)
else()
    target_compile_options(notehub-ui PRIVATE -Wall -Wextra -Wpedantic)
endif()
qt_add_executable(NoteHub WIN32 MACOSX_BUNDLE src/main.cpp resources.qrc)
target_link_libraries(NoteHub PRIVATE notehub-ui)
set_target_properties(NoteHub PROPERTIES RUNTIME_OUTPUT_DIRECTORY "${CMAKE_BINARY_DIR}/bin"
    MACOSX_BUNDLE_BUNDLE_NAME NoteHub MACOSX_BUNDLE_GUI_IDENTIFIER io.notehub.desktop)
find_program(GO_EXECUTABLE go REQUIRED)
set(CORE_FILENAME notehub-core${CMAKE_EXECUTABLE_SUFFIX})
add_custom_target(notehub-core
    COMMAND ${CMAKE_COMMAND} -E make_directory "${CMAKE_BINARY_DIR}/bin"
    COMMAND ${CMAKE_COMMAND} -E env CGO_ENABLED=0 ${GO_EXECUTABLE} build -trimpath
        -ldflags=-X=main.version=${PROJECT_VERSION}
        -o "${CMAKE_BINARY_DIR}/bin/${CORE_FILENAME}" ./cmd/notehub-core
    WORKING_DIRECTORY "${PROJECT_SOURCE_DIR}" VERBATIM)
add_dependencies(NoteHub notehub-core)
if(APPLE)
    add_custom_command(TARGET NoteHub POST_BUILD COMMAND ${CMAKE_COMMAND} -E copy_if_different
        "${CMAKE_BINARY_DIR}/bin/${CORE_FILENAME}" "$<TARGET_FILE_DIR:NoteHub>/${CORE_FILENAME}")
endif()
install(TARGETS NoteHub BUNDLE DESTINATION . RUNTIME DESTINATION .)
if(NOT APPLE)
    install(PROGRAMS "${CMAKE_BINARY_DIR}/bin/${CORE_FILENAME}" DESTINATION .)
endif()
qt_generate_deploy_app_script(TARGET NoteHub OUTPUT_SCRIPT deploy_script NO_UNSUPPORTED_PLATFORM_ERROR)
install(SCRIPT ${deploy_script})
if(BUILD_TESTING)
    add_executable(notehub-fake-core tests/FakeCore.cpp)
    target_link_libraries(notehub-fake-core PRIVATE Qt6::Core)
    add_executable(notehub-ui-tests tests/UiTests.cpp)
    add_dependencies(notehub-ui-tests notehub-fake-core)
    target_link_libraries(notehub-ui-tests PRIVATE notehub-ui Qt6::Test)
    add_test(NAME notehub-ui-tests COMMAND notehub-ui-tests -platform offscreen)
endif()
