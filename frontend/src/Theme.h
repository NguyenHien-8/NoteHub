#pragma once

#include <QString>

class QApplication;

namespace Theme {
void apply(QApplication &app, const QString &mode, const QString &fontFamily, int fontSize);
QString normalizedMode(const QString &mode);
}
