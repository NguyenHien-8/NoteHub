#include <QCoreApplication>
#include <QJsonArray>
#include <QJsonDocument>
#include <QJsonObject>
#include <QThread>
#include <iostream>

// Test-only peer deliberately splits records to exercise the QProcess parser.
int main(int argc, char **argv) {
    QCoreApplication app(argc, argv);
    std::string line;
    while (std::getline(std::cin, line)) {
        auto request = QJsonDocument::fromJson(QByteArray::fromStdString(line)).object();
        auto method = request["method"].toString();
        QJsonValue result;
        if (method == "hello")
            result = QJsonObject{{"protocol", 1}, {"version", "test"}, {"dataDir", "temporary test profile"}};
        else if (method == "metadata")
            result = QJsonObject{{"all", 1}, {"favorites", 0}, {"shared", 0}, {"tags", QJsonArray{}}};
        else if (method == "memos.list")
            result = QJsonObject{
                {"items",
                 QJsonArray{QJsonObject{
                     {"uid", "sample"},
                     {"content", "# Welcome to NoteHub\nA native Qt workspace. **Write clearly.** #Research"},
                     {"createdAt", "2026-10-03T08:00:00Z"},
                     {"attachments", QJsonArray{}}}}},
                {"cursor", ""}};
        else if (method == "calendar.month")
            result = QJsonArray{};
        else
            result = request["params"];
        auto bytes = QJsonDocument(QJsonObject{{"id", request["id"]}, {"result", result}})
                         .toJson(QJsonDocument::Compact) +
                     '\n';
        const auto split = bytes.size() / 2;
        std::cout.write(bytes.constData(), split);
        std::cout.flush();
        QThread::msleep(2);
        std::cout.write(bytes.constData() + split, bytes.size() - split);
        std::cout.flush();
    }
    return 0;
}
